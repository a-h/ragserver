package main

import (
	"context"
	"os"

	"github.com/a-h/ragserver/client"
	"github.com/a-h/ragserver/models"
)

type QueryCommand struct {
	RAGServerURL    string `help:"The URL of the RAG server." env:"RAG_SERVER_URL" default:"http://localhost:9020"`
	RAGServerAPIKey string `help:"The API key for the RAG server." env:"RAG_SERVER_API_KEY" default:""`
	NoContext       bool   `help:"Do not use context." env:"NO_CONTEXT" default:"false"`
	Query           string `help:"The query to send." short:"q"`
	LogLevel        string `help:"The log level to use." env:"LOG_LEVEL" default:"info"`
}

func (c QueryCommand) Run(ctx context.Context) (err error) {
	log := getLogger(c.LogLevel)
	if c.NoContext {
		log.Info("Querying without context")
	}

	rsc := client.New(c.RAGServerURL, c.RAGServerAPIKey)
	f := func(ctx context.Context, chunk []byte) error {
		_, err := os.Stdout.Write(chunk)
		return err
	}
	return rsc.QueryPost(ctx, models.QueryPostRequest{
		Text:      c.Query,
		NoContext: c.NoContext,
	}, f)
}
