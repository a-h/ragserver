package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/a-h/ragserver/client"
	"github.com/a-h/ragserver/models"
)

type ContextCommand struct {
	RAGServerURL    string `help:"The URL of the RAG server." env:"RAG_SERVER_URL" default:"http://localhost:9020"`
	RAGServerAPIKey string `help:"The API key for the RAG server." env:"RAG_SERVER_API_KEY" default:""`
	Text            string `help:"The text to send."`
	Pretty          bool   `help:"Pretty print the JSON output." default:"true"`
	LogLevel        string `help:"The log level to use." env:"LOG_LEVEL" default:"info"`
}

func (c ContextCommand) Run(ctx context.Context) (err error) {
	rsc := client.New(c.RAGServerURL, c.RAGServerAPIKey)
	resp, err := rsc.ContextPost(ctx, models.ContextPostRequest{
		Text: c.Text,
	})

	enc := json.NewEncoder(os.Stdout)
	if c.Pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(resp)
}
