package main

import (
	"context"
	"flag"
	"fmt"
	"iter"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/a-h/ragserver/client"
	"github.com/a-h/ragserver/models"
	"github.com/pluja/pocketbase"
)

func importCmd(ctx context.Context) (err error) {
	flags := flag.NewFlagSet("import", flag.ExitOnError)
	ragServerURL := flags.String("rag-server-url", "http://localhost:9020", "The URL of the RAG server.")
	pocketbaseURL := flags.String("pocketbase-url", "http://localhost:8080", "The URL of the Pocketbase server.")
	collectionName := flags.String("collection", "entities", "The name of the collection to export from.")
	expand := flags.String("expand", "", "The fields to expand.")
	level := flags.String("level", "info", "The log level to use, set to info for additional logs")
	if err = flags.Parse(os.Args[2:]); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	log := getLogger(*level)

	c := client.New(*ragServerURL)

	pbe := NewPocketbaseExporter(pocketbase.NewClient(*pocketbaseURL), *collectionName, *expand)
	for doc := range pbe.Export(ctx) {
		log.Info("importing document", slog.String("url", doc.URL))
		if log.Enabled(ctx, slog.LevelInfo) {
			fmt.Println(doc.Text)
		}
		resp, err := c.DocumentsPut(ctx, models.DocumentsPostRequest{
			Document: doc,
		})
		if err != nil {
			return fmt.Errorf("failed to put document: %w", err)
		}
		log.Info("document imported", slog.String("url", doc.URL), slog.Int64("id", resp.ID))
	}
	return pbe.Error
}

func NewPocketbaseExporter(client *pocketbase.Client, collection, expand string) *PocketbaseExporter {
	return &PocketbaseExporter{
		client:     client,
		collection: collection,
		expand:     expand,
		PageSize:   10,
		Error:      nil,
	}
}

type PocketbaseExporter struct {
	client     *pocketbase.Client
	collection string
	expand     string
	PageSize   int
	Error      error
}

func (p *PocketbaseExporter) Export(ctx context.Context) iter.Seq[models.Document] {
	var page int
	return func(yield func(models.Document) bool) {
		for {
			if ctx.Err() != nil {
				return
			}
			if p.Error != nil {
				return
			}
			page++
			response, err := p.client.List(p.collection, pocketbase.ParamsList{
				Page:   page,
				Size:   p.PageSize,
				Sort:   "-created",
				Expand: p.expand,
			})
			if err != nil {
				p.Error = err
				return
			}
			if len(response.Items) == 0 {
				return
			}
			for _, item := range response.Items {
				if !yield(p.createDocument(item)) {
					return
				}
			}
		}
	}
}

func useItemOrDefault(item map[string]any, keys []string, defaultValue string) string {
	for _, key := range keys {
		if value, ok := item[key].(string); ok {
			return value
		}
	}
	return defaultValue
}

func (p *PocketbaseExporter) createDocument(item map[string]any) (d models.Document) {
	d.URL = useItemOrDefault(item, []string{"url"}, fmt.Sprintf("%s/%s", url.PathEscape(p.collection), url.PathEscape(item["id"].(string))))
	d.Title = useItemOrDefault(item, []string{"title", "name"}, "Untitled")
	d.Text = p.getText(item)
	d.Summary = useItemOrDefault(item, []string{"summary"}, "")
	return
}

func (p *PocketbaseExporter) getText(item map[string]any) string {
	var sb strings.Builder
	for key, value := range item {
		sb.WriteString(fmt.Sprintf("%s: %s\n", key, getTextFromValue(value)))
	}
	return sb.String()
}

func getTextFromValue(value any) string {
	switch v := value.(type) {
	case map[string]any:
		var sb strings.Builder
		for key, item := range v {
			sb.WriteString(fmt.Sprintf("%s: %s\n", key, getTextFromValue(item)))
		}
		return sb.String()
	case string:
		return v
	case []any:
		var sb strings.Builder
		for i, item := range v {
			sb.WriteString(fmt.Sprintf("%d: %s\n", i, getTextFromValue(item)))
		}
		return sb.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}
