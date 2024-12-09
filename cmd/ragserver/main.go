package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	_ "embed"
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
	case "import":
		err = importCmd(ctx)
	case "query":
		err = queryCmd(ctx)
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

  ragserver import
    - Import documents into a RAG server.

  ragserver query
    - Query the RAG store and LLM.

  ragserver version
    - Print the version of the RAG server.`)
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
