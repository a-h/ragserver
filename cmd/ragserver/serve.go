package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/a-h/ragserver/db"
	documentspost "github.com/a-h/ragserver/handlers/documents/post"
	querypost "github.com/a-h/ragserver/handlers/query/post"
	"github.com/rqlite/gorqlite"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
)

type ServeCommand struct {
	RqliteURL      string `help:"The URL of the rqlite server." env:"RQLITE_URL" default:"http://localhost:4001"`
	OllamaURL      string `help:"The URL of the Ollama server." env:"OLLAMA_URL" default:"http://127.0.0.1:11434/"`
	EmbeddingModel string `help:"The model to use for embeddings." env:"EMBEDDING_MODEL" default:"nomic-embed-text"`
	ChatModel      string `help:"The model to chat with." env:"CHAT_MODEL" default:"mistral-nemo"`
	SystemPrompt   string `help:"The system prompt to use." env:"SYSTEM_PROMPT" default:""`
	UserPrompt     string `help:"The user prompt to use." env:"USER_PROMPT" default:""`
	MaxContextDocs int    `help:"The maximum number of context documents to use." env:"MAX_CONTEXT_DOCS" default:"5"`
	ListenAddr     string `help:"The address to listen on." env:"LISTEN_ADDR" default:"localhost:9020"`
	LogLevel       string `help:"The log level to use." env:"LOG_LEVEL" default:"info"`
}

const systemPrompt = `You are a trusted advisor that doesn't make up answers. You are provided with context and a question. You always use the context to answer the question. If you don't know the answer, you say that you don't know, and don't try to make up an answer.

You respect the user's time and don't provide unnecessary information. You are succinct and to the point.`

const userPrompt = `Here is the context you need to answer the question:

%s

Please provide a succint response to: %s`

func readFileOrDefault(filename, defaultContent string) (string, error) {
	if filename == "" {
		return defaultContent, nil
	}
	contents, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filename, err)
	}
	return string(contents), nil
}

func (c ServeCommand) Run(ctx context.Context) (err error) {
	log := getLogger(c.LogLevel)
	systemPrompt, err := readFileOrDefault(c.SystemPrompt, systemPrompt)
	if err != nil {
		return fmt.Errorf("failed to read system prompt: %w", err)
	}
	userPrompt, err := readFileOrDefault(c.UserPrompt, userPrompt)
	if err != nil {
		return fmt.Errorf("failed to read user prompt: %w", err)
	}
	pf := func(q, context string) (string, error) {
		return fmt.Sprintf(userPrompt, context, q), nil
	}
	if _, err = pf("hello", "world"); err != nil {
		return fmt.Errorf("invalid prompt template: %w", err)
	}

	log.Info("connecting to database", slog.String("url", c.RqliteURL))
	databaseURL, err := db.ParseRqliteURL(c.RqliteURL)
	if err != nil {
		return fmt.Errorf("failed to parse rqlite URL: %w", err)
	}
	log.Info("opening database connection", slog.String("url", databaseURL.DataSourceName()))
	conn, err := gorqlite.Open(databaseURL.DataSourceName())
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}
	defer conn.Close()
	queries := db.New(conn)

	log.Info("migrating database schema", slog.String("url", databaseURL.MigrateDatabaseURL()))
	if err = db.Migrate(databaseURL); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Info("creating LLM clients")
	httpClient := &http.Client{}
	ec, err := ollama.New(
		ollama.WithModel(c.EmbeddingModel),
		ollama.WithHTTPClient(httpClient),
		ollama.WithServerURL(c.OllamaURL))
	if err != nil {
		return fmt.Errorf("failed to create embedder: %w", err)
	}
	emb, err := embeddings.NewEmbedder(ec)
	if err != nil {
		return fmt.Errorf("failed to create embedder: %w", err)
	}

	llmc, err := ollama.New(
		ollama.WithModel(c.ChatModel),
		ollama.WithHTTPClient(httpClient),
		ollama.WithServerURL(c.OllamaURL))
	if err != nil {
		return fmt.Errorf("failed to create LLM: %w", err)
	}

	mux := http.NewServeMux()

	dah := documentspost.New(log, emb, queries)
	mux.Handle("POST /documents", dah)

	qph := querypost.New(log, emb, llmc, queries, c.MaxContextDocs, systemPrompt, pf)
	mux.Handle("POST /query", qph)

	log.Info("listening", slog.String("addr", c.ListenAddr))
	return http.ListenAndServe(c.ListenAddr, mux)
}
