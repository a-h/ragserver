package db_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/a-h/ragserver/db"
	"github.com/google/go-cmp/cmp"
	"github.com/rqlite/gorqlite"
)

var initOnce sync.Once
var conn *gorqlite.Connection

func initConnection() (err error) {
	url := "http://admin:secret@localhost:4001"
	databaseURL, err := db.ParseRqliteURL(url)
	if err != nil {
		return fmt.Errorf("failed to parse rqlite URL: %w", err)
	}
	initOnce.Do(func() {
		conn, err = gorqlite.Open(databaseURL.DataSourceName())
		if err != nil {
			err = fmt.Errorf("failed to open connection: %w", err)
			return
		}
		if err = db.Migrate(databaseURL); err != nil {
			err = fmt.Errorf("failed to migrate database: %w", err)
			return
		}
	})
	return err
}

const testPartitionName = "test-partition"

func TestDocument(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	if err := initConnection(); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	q := db.New(conn)
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	article1ID := db.DocumentID{
		Partition: testPartitionName,
		URL:       "https://example.com/article1",
	}

	article1 := db.Document{
		DocumentID:    article1ID,
		Title:         "Example Article",
		Text:          "This is an example article.",
		Summary:       "An example article.",
		CreatedAt:     now,
		LastUpdatedAt: now,
	}
	article1Chunks := []db.Chunk{
		createChunk("Chunk 0"),
		createChunk("Chunk 1"),
		createChunk("Chunk 2"),
		createChunk("Chunk 3"),
	}

	t.Run("Can delete previous records", func(t *testing.T) {
		id, err := q.DocumentPut(ctx, db.DocumentPutArgs{
			Document: article1,
			Chunks:   article1Chunks,
		})
		if err != nil {
			t.Fatalf("failed to insert document: %v", err)
		}
		if id == 0 {
			t.Errorf("expected a non-zero row ID")
		}

		err = q.DocumentDelete(ctx, article1ID)
		if err != nil {
			t.Fatalf("failed to delete document: %v", err)
		}

		_, ok, err := q.DocumentGet(ctx, article1ID)
		if err != nil {
			t.Fatalf("failed to get document: %v", err)
		}
		if ok {
			t.Fatalf("document found")
		}
	})

	t.Run("Can insert and retrieve new records", func(t *testing.T) {
		id, err := q.DocumentPut(ctx, db.DocumentPutArgs{
			Document: article1,
			Chunks:   article1Chunks,
		})
		if err != nil {
			t.Fatalf("failed to insert document: %v", err)
		}
		if id == 0 {
			t.Errorf("expected a non-zero row ID")
		}

		doc, ok, err := q.DocumentGet(ctx, article1ID)
		if err != nil {
			t.Fatalf("failed to get document: %v", err)
		}
		if !ok {
			t.Fatalf("document not found")
		}
		if diff := cmp.Diff(article1, doc); diff != "" {
			t.Fatalf("unexpected document: %v", diff)
		}
	})

	t.Run("Can upsert over an existing record", func(t *testing.T) {
		updatedDate := now.Add(time.Hour)
		updated := db.Document{
			DocumentID:    article1ID,
			Title:         "Updated Article",
			Text:          "This is an updated example article.",
			Summary:       "An example article, updated.",
			CreatedAt:     now,
			LastUpdatedAt: updatedDate,
		}
		// Remove a chunk.
		article1Chunks = article1Chunks[:len(article1Chunks)-1]

		id, err := q.DocumentPut(ctx, db.DocumentPutArgs{
			Document: updated,
			Chunks:   article1Chunks,
		})
		if err != nil {
			t.Fatalf("failed to upsert document: %v", err)
		}
		if id == 0 {
			t.Errorf("expected a non-zero row ID")
		}

		doc, ok, err := q.DocumentGet(ctx, article1ID)
		if err != nil {
			t.Fatalf("failed to get document: %v", err)
		}
		if !ok {
			t.Fatalf("document not found")
		}
		if diff := cmp.Diff(updated, doc); diff != "" {
			t.Fatalf("unexpected document: %v", diff)
		}
	})
}

func createChunk(s string) (chunk db.Chunk) {
	chunk.Text = s
	chunk.Embedding = make([]float32, 768)
	for i := 0; i < 768; i++ {
		chunk.Embedding[i] = float32(i)
	}
	return chunk
}
