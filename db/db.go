package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rqlite/gorqlite"
)

func New(conn *gorqlite.Connection) *Queries {
	return &Queries{
		conn: conn,
	}
}

type Queries struct {
	conn *gorqlite.Connection
}

type DocumentID struct {
	Partition string
	URL       string
}

func (d DocumentID) String() string {
	return fmt.Sprintf("%s:%s", d.Partition, d.URL)
}

func newDocumentUpsertRowIDArgs(id DocumentID, title, summary string, createdAt, lastUpdatedAt time.Time) documentUpsertRowIDArgs {
	return documentUpsertRowIDArgs{
		DocumentID:    id,
		Title:         title,
		Summary:       summary,
		CreatedAt:     createdAt,
		LastUpdatedAt: lastUpdatedAt,
	}
}

type documentUpsertRowIDArgs struct {
	DocumentID
	Title         string
	Summary       string
	CreatedAt     time.Time
	LastUpdatedAt time.Time
}

func (q *Queries) documentUpsertRowID(ctx context.Context, args documentUpsertRowIDArgs) (rowID int64, err error) {
	stmt := gorqlite.ParameterizedStatement{
		Query: `insert into document (id, partition, url, title, summary, created_at, last_updated_at)
values (?, ?, ?, ?, ?, ?, ?)
on conflict(id) do update
set
    partition = excluded.partition,
    url = excluded.url,
    title = excluded.title,
    summary = excluded.summary,
    last_updated_at = excluded.last_updated_at
`,
		Arguments: []any{args.DocumentID.String(), args.Partition, args.URL, args.Title, args.Summary, args.CreatedAt, args.LastUpdatedAt},
	}
	_, err = q.conn.WriteOneParameterizedContext(ctx, stmt)
	if err != nil {
		return 0, err
	}

	// Read the row ID.
	stmt = gorqlite.ParameterizedStatement{
		Query:     `select rowid from document where id = ?`,
		Arguments: []any{args.DocumentID.String()},
	}
	result, err := q.conn.QueryOneParameterizedContext(ctx, stmt)
	if err != nil {
		return 0, err
	}
	if !result.Next() {
		return 0, fmt.Errorf("expected a row ID")
	}
	err = result.Scan(&rowID)
	return rowID, err
}

type Document struct {
	DocumentID
	Title         string
	Text          string
	Summary       string
	CreatedAt     time.Time
	LastUpdatedAt time.Time
}

type DocumentPutArgs struct {
	Document Document
	Chunks   []Chunk
}

type Chunk struct {
	Text      string
	Embedding []float32
}

func (q *Queries) DocumentPut(ctx context.Context, args DocumentPutArgs) (id int64, err error) {
	id, err = q.documentUpsertRowID(ctx, newDocumentUpsertRowIDArgs(args.Document.DocumentID, args.Document.Title, args.Document.Summary, args.Document.CreatedAt, args.Document.LastUpdatedAt))
	if err != nil {
		return id, fmt.Errorf("failed to upsert document row id: %w", err)
	}
	if id == 0 {
		return id, fmt.Errorf("expected a non-zero row ID")
	}

	statements := make([]gorqlite.ParameterizedStatement, len(args.Chunks)+2)
	for chunkIndex, chunk := range args.Chunks {
		embeddingJSON, err := json.Marshal(chunk.Embedding)
		if err != nil {
			return id, fmt.Errorf("failed to marshal embedding: %w", err)
		}
		statements[chunkIndex] = gorqlite.ParameterizedStatement{
			Query:     `insert or replace into document_chunk_vec (document_rowid, partition, idx, text, embedding) values (?, ?, ?, ?, ?)`,
			Arguments: []any{id, args.Document.Partition, chunkIndex, chunk.Text, string(embeddingJSON)},
		}
	}
	// Delete excess rows.
	statements[len(statements)-2] = gorqlite.ParameterizedStatement{
		Query:     `delete from document_chunk_vec where document_rowid = ? and idx > ?`,
		Arguments: []any{id, len(args.Chunks) - 1},
	}
	// Insert into the FTS table.
	statements[len(statements)-1] = gorqlite.ParameterizedStatement{
		Query:     `insert or replace into document_fts (rowid, partition, url, title, text, summary) values (?, ?, ?, ?, ?, ?)`,
		Arguments: []any{id, args.Document.Partition, args.Document.URL, args.Document.Title, args.Document.Text, args.Document.Summary},
	}
	if _, err = q.conn.WriteParameterizedContext(ctx, statements); err != nil {
		return id, err
	}
	return id, nil
}

func (q *Queries) DocumentDelete(ctx context.Context, args DocumentID) (err error) {
	statements := []gorqlite.ParameterizedStatement{
		{
			Query:     `delete from document_chunk_vec where document_rowid in (select rowid from document where partition = ? and url = ?)`,
			Arguments: []any{args.Partition, args.URL},
		},
		{
			Query:     `delete from document_fts where rowid in (select rowid from document where partition = ? and url = ?)`,
			Arguments: []any{args.Partition, args.URL},
		},
		{
			Query:     `delete from document where partition = ? and url = ?`,
			Arguments: []any{args.Partition, args.URL},
		},
	}
	if _, err = q.conn.WriteParameterizedContext(ctx, statements); err != nil {
		return err
	}
	return nil
}

func (q *Queries) DocumentGet(ctx context.Context, args DocumentID) (doc Document, ok bool, err error) {
	stmt := gorqlite.ParameterizedStatement{
		Query:     "select document.partition, document.url, document.title, document_fts.text, document.summary, document.created_at, document.last_updated_at from document_fts inner join document on document.rowid = document_fts.rowid where document_fts.partition = ? and document_fts.url = ?",
		Arguments: []any{args.Partition, args.URL},
	}
	result, err := q.conn.QueryOneParameterizedContext(ctx, stmt)
	if err != nil {
		return Document{}, false, err
	}
	if !result.Next() {
		return Document{}, false, nil
	}
	if err = result.Scan(&doc.Partition, &doc.URL, &doc.Title, &doc.Text, &doc.Summary, &doc.CreatedAt, &doc.LastUpdatedAt); err != nil {
		return Document{}, false, err
	}
	return doc, true, nil
}

type DocumentSelectNearestArgs struct {
	Partition string
	Embedding []float32
	Limit     int
}

type DocumentSelectNearestResult struct {
	RowID     int64
	Partition string
	Index     int64
	Text      string
	Embedding []float32
	Distance  float64
	URL       string
	Title     string
	Summary   string
}

/*
select
  article_id,
  headline,
  news_desk,
  word_count,
  url,
  pub_date,
  distance
from vec_articles
where headline_embedding match lembed('pandemic')
  and k = 8
  and year = 2020
  and news_desk in ('Sports', 'Business')
  and word_count between 500 and 1000;
*/

func (q *Queries) DocumentNearest(ctx context.Context, args DocumentSelectNearestArgs) (docs []DocumentSelectNearestResult, err error) {
	inputEmbeddingJSON, err := json.Marshal(args.Embedding)
	if err != nil {
		return docs, fmt.Errorf("failed to marshal input embedding: %w", err)
	}
	stmt := gorqlite.ParameterizedStatement{
		Query: `with limited_dcv as (
  select document_rowid, partition, idx, text, embedding, distance
  from document_chunk_vec
  where partition = ? and embedding match ?
  order by distance asc
  limit ?
)
select 
  ld.document_rowid,
  ld.partition,
  ld.idx,
  ld.text,
  vec_to_json(ld.embedding),
  ld.distance,
  d.url,
  d.title,
  d.summary
from limited_dcv ld
left join document d on d.rowid = ld.document_rowid;`,
		Arguments: []any{args.Partition, string(inputEmbeddingJSON), args.Limit},
	}
	result, err := q.conn.QueryOneParameterizedContext(ctx, stmt)
	if err != nil {
		return docs, err
	}
	for result.Next() {
		var doc DocumentSelectNearestResult
		var embeddingJSON string
		if err = result.Scan(&doc.RowID, &doc.Partition, &doc.Index, &doc.Text, &embeddingJSON, &doc.Distance, &doc.URL, &doc.Title, &doc.Summary); err != nil {
			return docs, err
		}
		if err = json.Unmarshal([]byte(embeddingJSON), &doc.Embedding); err != nil {
			return docs, fmt.Errorf("failed to unmarshal embedding: %w", err)
		}
		docs = append(docs, doc)
	}
	return docs, nil
}
