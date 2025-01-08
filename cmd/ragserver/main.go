package main

import (
	"context"
	"log/slog"
	"os"

	_ "embed"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Serve   ServeCommand   `cmd:"serve" help:"Start the RAG server."`
	Import  ImportCommand  `cmd:"import" help:"Import documents into a RAG server."`
	Context ContextCommand `cmd:"context" help:"Get similar documents for a piece of text."`
	Chat    ChatCommand    `cmd:"chat" help:"Chat with the RAG server."`
	Query   QueryCommand   `cmd:"query" help:"Query the RAG store and LLM."`
	Version VersionCommand `cmd:"version" help:"Print the version of the RAG server."`
}

func main() {
	var cli CLI
	ctx := context.Background()
	kctx := kong.Parse(&cli, kong.UsageOnError(), kong.BindTo(ctx, (*context.Context)(nil)))
	if err := kctx.Run(); err != nil {
		log := getLogger("error")
		log.Error("error", slog.Any("error", err))
		os.Exit(1)
	}
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
