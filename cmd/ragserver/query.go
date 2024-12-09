package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/a-h/ragserver/client"
	"github.com/a-h/ragserver/models"
)

func queryCmd(ctx context.Context) (err error) {
	flags := flag.NewFlagSet("query", flag.ExitOnError)
	ragServerURL := flags.String("rag-server-url", "http://localhost:9020", "The URL of the RAG server.")
	nocontext := flags.Bool("no-context", false, "Do not use context.")
	query := flags.String("q", "", "The query to send.")
	level := flags.String("level", "info", "The log level to use, set to info for additional logs")
	if err = flags.Parse(os.Args[2:]); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	log := getLogger(*level)
	if *nocontext {
		log.Info("Querying without context")
	}

	c := client.New(*ragServerURL)
	f := func(ctx context.Context, chunk []byte) error {
		_, err := os.Stdout.Write(chunk)
		return err
	}
	return c.QueryPost(ctx, models.QueryPostRequest{
		Text:      *query,
		NoContext: *nocontext,
	}, f)
}
