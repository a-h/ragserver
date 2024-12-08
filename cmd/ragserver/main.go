package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	_ "embed"

	"github.com/a-h/ragserver/db"
	documentspost "github.com/a-h/ragserver/handlers/documents/post"
	querypost "github.com/a-h/ragserver/handlers/query/post"
	"github.com/rqlite/gorqlite"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
)

var version string

func main() {
	if len(os.Args) < 2 || os.Args[1] == "--help" || os.Args[1] == "-h" {
		printUsage()
		os.Exit(1)
	}
	ctx := context.Background()
	var err error

	switch os.Args[1] {
	case "version":
		fallthrough
	case "-v":
		fallthrough
	case "--version":
		fallthrough
	case "-version":
		fmt.Println(version)
	case "serve":
		err = serveCmd(ctx)
	default:
		fmt.Printf("unknown command %q\n", os.Args[1])
		fmt.Println()
		printUsage()
	}

	if err != nil {
		log := getLogger("error")
		log.Error("error", slog.Any("error", err))
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`ragserver is a JSON API that provides Retrieval Augmented Generation for an LLM.

Usage:

  ragserver serve
    - Start the RAG server.

  ragserver version
    - Print the version of the RAG server.`)
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

func getLogger(level string) *slog.Logger {
	ll := slog.LevelInfo
	switch level {
	case "debug":
		ll = slog.LevelDebug
	case "info":
		ll = slog.LevelInfo
	case "warn":
		ll = slog.LevelWarn
	case "error":
		ll = slog.LevelError
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: ll,
	}))
}
