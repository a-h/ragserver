package main

import (
	"context"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-h/ragserver/client"
	"github.com/a-h/ragserver/models"
	"github.com/pluja/pocketbase"
	"github.com/tmc/langchaingo/documentloaders"
	"gopkg.in/yaml.v3"
)

type ImportCommand struct {
	RAGServerURL    string `help:"The URL of the RAG server." env:"RAG_SERVER_URL" default:"http://localhost:9020"`
	RAGServerAPIKey string `help:"The API key for the RAG server." env:"RAG_SERVER_API_KEY" default:""`
	PocketbaseURL   string `help:"The URL of the Pocketbase server." env:"POCKETBASE_URL" default:"http://localhost:8080"`
	ID              string `help:"The ID of the document to import if you just want to import a single doc." env:"ID" default:""`
	Collection      string `help:"The name of the collection to export from." env:"COLLECTION" default:"entities"`
	Expand          string `help:"The fields to expand." env:"EXPAND" default:""`
	Files           string `help:"Comma separated list of fields that contain Pocketbase file references." env:"FILES" default:""`
	DryRun          bool   `help:"Do not actually import the documents." env:"DRY_RUN" default:"false"`
	LogLevel        string `help:"The log level to use." env:"LOG_LEVEL" default:"info"`
}

func (c ImportCommand) Run(ctx context.Context) (err error) {
	log := getLogger(c.LogLevel)

	rsc := client.New(c.RAGServerURL, c.RAGServerAPIKey)

	pbe := NewPocketbaseExporter(c.PocketbaseURL, pocketbase.NewClient(c.PocketbaseURL), c.Collection, c.Expand, c.Files)
	for doc := range pbe.Export(ctx) {
		if c.ID != "" && doc.ID != c.ID {
			continue
		}
		log.Info("importing document", slog.String("url", doc.Document.URL))
		if log.Enabled(ctx, slog.LevelInfo) {
			fmt.Println(doc.Document.Text)
		}
		if c.DryRun {
			log.Info("skipping document import in dry run mode", slog.String("url", doc.Document.URL))
			continue
		}
		resp, err := rsc.DocumentsPut(ctx, models.DocumentsPostRequest{
			Document: doc.Document,
		})
		if err != nil {
			return fmt.Errorf("failed to put document: %w", err)
		}
		log.Info("document imported", slog.String("url", doc.Document.URL), slog.Int64("id", resp.ID))
	}
	return pbe.Error
}

func NewPocketbaseExporter(baseURL string, client *pocketbase.Client, collection, expand, files string) *PocketbaseExporter {
	return &PocketbaseExporter{
		baseURL:    baseURL,
		client:     client,
		collection: collection,
		expand:     expand,
		files:      strings.Split(files, ","),
		PageSize:   10,
		Error:      nil,
	}
}

type PocketbaseExporter struct {
	// baseURL for downloading files, e.g. http://localhost:8090
	baseURL    string
	client     *pocketbase.Client
	collection string
	expand     string
	files      []string
	PageSize   int
	Error      error
}

func (p *PocketbaseExporter) Export(ctx context.Context) iter.Seq[ExportedDocument] {
	var page int
	return func(yield func(ExportedDocument) bool) {
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
				if !yield(p.createDocument(ctx, item)) {
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

type ExportedDocument struct {
	ID       string
	Document models.Document
}

func (p *PocketbaseExporter) createDocument(ctx context.Context, item map[string]any) (ed ExportedDocument) {
	ed.ID = item["id"].(string)
	ed.Document.URL = useItemOrDefault(item, []string{"url"}, fmt.Sprintf("%s/%s", url.PathEscape(p.collection), url.PathEscape(item["id"].(string))))
	ed.Document.Title = useItemOrDefault(item, []string{"title", "name"}, "Untitled")
	recursivelyApplyExpandedFields(item)
	recursivelyRemoveKeys(item, []string{"id", "collectionId", "collectionName", "created", "updated"})
	ed.Document.Summary = useItemOrDefault(item, []string{"summary"}, "")

	sb := new(strings.Builder)
	_ = yaml.NewEncoder(sb).Encode(item)

	for _, fileFieldName := range p.files {
		if ctx.Err() != nil {
			return
		}
		fileNames, fileNamesFieldExists := item[fileFieldName].([]any)
		if !fileNamesFieldExists || len(fileNames) == 0 {
			continue
		}
		for _, fileName := range fileNames {
			// Check if the file name is a string.
			fileName, ok := fileName.(string)
			if !ok {
				p.Error = fmt.Errorf("file name is not a string")
				continue
			}
			if !strings.EqualFold(filepath.Ext(fileName), ".pdf") {
				continue
			}
			// Get the file text.
			fileText, err := p.getPDFText(ctx, p.collection, ed.ID, fileName)
			if err != nil {
				p.Error = fmt.Errorf("failed to get file text: %w", err)
				continue
			}
			sb.WriteString(fileText)
		}
	}

	ed.Document.Text = sb.String()

	return
}

func (p *PocketbaseExporter) getPDFText(ctx context.Context, collection, id, filename string) (string, error) {
	// Start download.
	downloadURL, err := createURL(p.baseURL, "api", "files", collection, id, filename)
	if err != nil {
		return "", fmt.Errorf("failed to create download URL: %w", err)
	}
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// Create temp file.
	pdfFile, err := os.CreateTemp("", "rag-import-*.pdf")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer pdfFile.Close()
	defer os.Remove(pdfFile.Name())

	// Write the HTTP response to the file.
	fileSize, err := io.Copy(pdfFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Read the PDF text.
	pdf := documentloaders.NewPDF(pdfFile, fileSize)
	docs, err := pdf.Load(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load PDF: %w", err)
	}

	// Create the output.
	var sb strings.Builder
	for _, doc := range docs {
		sb.WriteString(doc.PageContent)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func createURL(baseURL string, pathSegments ...string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse baseURL: %w", err)
	}
	u.Path = strings.Join(pathSegments, "/")
	return u.String(), nil
}

func applyExpandedFields(data map[string]any) (changed bool) {
	for key, value := range data {
		if key == "expand" {
			expandMap, ok := value.(map[string]any)
			if !ok {
				continue
			}

			// Check parent keys for matches in expand.
			for parentKey := range data {
				if parentKey == "expand" {
					continue
				}
				if expandedValue, found := expandMap[parentKey]; found {
					data[parentKey] = expandedValue
					changed = true
				}
			}

			// Remove expand key.
			delete(data, "expand")
			changed = true
		} else if nestedMap, ok := value.(map[string]any); ok {
			// Recurse into nested maps.
			if applyExpandedFields(nestedMap) {
				changed = true
			}
		} else if nestedSlice, ok := value.([]any); ok {
			// Recurse into slices.
			for _, item := range nestedSlice {
				if itemMap, isMap := item.(map[string]any); isMap {
					if applyExpandedFields(itemMap) {
						changed = true
					}
				}
			}
		}
	}

	return changed
}

func recursivelyApplyExpandedFields(data map[string]any) {
	for {
		if changesMade := applyExpandedFields(data); !changesMade {
			return
		}
	}
}

func recursivelyRemoveKeys(item any, keys []string) {
	switch item := item.(type) {
	case map[string]any:
		for _, key := range keys {
			delete(item, key)
		}
		var emptyKeys []string
		for k, v := range item {
			switch v := v.(type) {
			case map[string]any:
				if len(v) == 0 {
					emptyKeys = append(emptyKeys, k)
				}
			case []any:
				if len(v) == 0 {
					emptyKeys = append(emptyKeys, k)
				}
			case string:
				if v == "" {
					emptyKeys = append(emptyKeys, k)
				}
			}
			recursivelyRemoveKeys(v, keys)
		}
		for _, key := range emptyKeys {
			delete(item, key)
		}
	case []any:
		for _, value := range item {
			recursivelyRemoveKeys(value, keys)
		}
	}
}
