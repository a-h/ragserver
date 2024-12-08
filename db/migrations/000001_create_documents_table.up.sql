create table document (
  -- There's an implicit rowid column.
  -- But we also add a primary key column to allow upserts with predictable
  -- primary key values, e.g. partition:url.
  id text primary key,

  -- Partition is used to shard the data between users.
  partition text not null,
  url text not null,

  title text not null,
  summary text not null,
  created_at text not null,
  last_updated_at text not null
);

create virtual table document_chunk_vec using vec0(
  -- Each chunk belongs to a document.
  document_rowid integer not null references document(rowid),

  -- Content is sharded based on partition.
  partition text not null partition key,

  -- Metadata columns, can appear in `WHERE` clause of KNN queries.
  idx integer not null, -- Index of the chunk in the doc.
  
  -- Auxiliary column, unindexed.
  +text text not null, -- The text of the chunk.
  embedding float[768]
);

-- Create full-text search table for documents.
create virtual table document_fts using fts5(
    partition unindexed,
    url unindexed,
    title,
    text,
    summary
);
