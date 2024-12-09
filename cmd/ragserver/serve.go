package main

import (
	"context"
	"flag"
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

func serveCmd(ctx context.Context) (err error) {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	ollamaURL := flags.String("ollama-url", "http://127.0.0.1:11434/", "The URL of the Ollama server.")
	embeddingModel := flags.String("embedding-model", "nomic-embed-text", "The model to use for embeddings.")
	chatModel := flags.String("chat-model", "mistral-nemo", "The model to chat with.")
	systemPromptFlag := flags.String("system-prompt", "", "The system prompt to use.")
	userPromptFlag := flags.String("user-prompt", "", "The user prompt to use.")
	maxContextDocs := flags.Int("max-context-docs", 5, "The maximum number of context documents to use.")
	listenAddr := flags.String("listen-addr", "localhost:9020", "The address to listen on.")
	level := flags.String("level", "info", "The log level to use, set to info for additional logs")
	if err = flags.Parse(os.Args[2:]); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	log := getLogger(*level)
	systemPrompt, err := readFileOrDefault(*systemPromptFlag, systemPrompt)
	if err != nil {
		return fmt.Errorf("failed to read system prompt: %w", err)
	}
	userPrompt, err := readFileOrDefault(*userPromptFlag, userPrompt)
	if err != nil {
		return fmt.Errorf("failed to read user prompt: %w", err)
	}
	pf := func(q, context string) (string, error) {
		return fmt.Sprintf(userPrompt, context, q), nil
	}
	if _, err = pf("hello", "world"); err != nil {
		return fmt.Errorf("invalid prompt template: %w", err)
	}

	databaseURL := db.URL{
		User:     "admin",
		Password: "secret",
		Host:     "localhost",
		Port:     4001,
		Secure:   false,
	}

	log.Info("connecting to database")
	conn, err := gorqlite.Open(databaseURL.DataSourceName())
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}
	defer conn.Close()
	queries := db.New(conn)

	log.Info("migrating database schema")
	if err = db.Migrate(databaseURL); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Info("creating LLM clients")
	httpClient := &http.Client{}
	ec, err := ollama.New(
		ollama.WithModel(*embeddingModel),
		ollama.WithHTTPClient(httpClient),
		ollama.WithServerURL(*ollamaURL))
	if err != nil {
		return fmt.Errorf("failed to create embedder: %w", err)
	}
	emb, err := embeddings.NewEmbedder(ec)
	if err != nil {
		return fmt.Errorf("failed to create embedder: %w", err)
	}

	llmc, err := ollama.New(
		ollama.WithModel(*chatModel),
		ollama.WithHTTPClient(httpClient),
		ollama.WithServerURL(*ollamaURL))
	if err != nil {
		return fmt.Errorf("failed to create LLM: %w", err)
	}

	mux := http.NewServeMux()

	dah := documentspost.New(log, emb, queries)
	mux.Handle("POST /documents", dah)

	qph := querypost.New(log, emb, llmc, queries, *maxContextDocs, systemPrompt, pf)
	mux.Handle("POST /query", qph)

	log.Info("listening", slog.String("addr", *listenAddr))
	return http.ListenAndServe(*listenAddr, mux)
}
