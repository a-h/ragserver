package main

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"net/url"
	"strings"

	"github.com/a-h/ragserver/client"
	"github.com/a-h/ragserver/models"
	"github.com/pluja/pocketbase"
)

type ImportCommand struct {
	RAGServerURL    string `help:"The URL of the RAG server." env:"RAG_SERVER_URL" default:"http://localhost:9020"`
	RAGServerAPIKey string `help:"The API key for the RAG server." env:"RAG_SERVER_API_KEY" default:""`
	PocketbaseURL   string `help:"The URL of the Pocketbase server." env:"POCKETBASE_URL" default:"http://localhost:8080"`
	Collection      string `help:"The name of the collection to export from." env:"COLLECTION" default:"entities"`
	Expand          string `help:"The fields to expand." env:"EXPAND" default:""`
	LogLevel        string `help:"The log level to use." env:"LOG_LEVEL" default:"info"`
}

func (c ImportCommand) Run(ctx context.Context) (err error) {
	log := getLogger(c.LogLevel)

	rsc := client.New(c.RAGServerURL, c.RAGServerAPIKey)

	pbe := NewPocketbaseExporter(pocketbase.NewClient(c.PocketbaseURL), c.Collection, c.Expand)
	for doc := range pbe.Export(ctx) {
		log.Info("importing document", slog.String("url", doc.URL))
		if log.Enabled(ctx, slog.LevelInfo) {
			fmt.Println(doc.Text)
		}
		resp, err := rsc.DocumentsPut(ctx, models.DocumentsPostRequest{
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
	d.Text = getTextFromValue(0, item)
	d.Summary = useItemOrDefault(item, []string{"summary"}, "")
	return
}

func getTextFromValue(depth int, value any) string {
	switch v := value.(type) {
	case map[string]any:
		var sb strings.Builder
		for key, item := range v {
			sb.WriteString(strings.Repeat("#", depth+1))
			sb.WriteString(" ")
			sb.WriteString(key)
			sb.WriteString("\n\n")
			sb.WriteString(getTextFromValue(depth+1, item))
			sb.WriteString("\n\n")
		}
		return sb.String()
	case float64:
		return fmt.Sprintf("%f", v)
	case string:
		return v
	case []any:
		var sb strings.Builder
		for _, item := range v {
			sb.WriteString(" - ")
			sb.WriteString(getTextFromValue(depth+1, item))
			sb.WriteString("\n")
		}
		return sb.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}
