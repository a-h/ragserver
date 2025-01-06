package post

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/a-h/ragserver/auth"
	"github.com/a-h/ragserver/db"
	"github.com/a-h/ragserver/models"
	"github.com/a-h/respond"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
)

func New(log *slog.Logger, embedder embeddings.Embedder, llm llms.Model, queries *db.Queries, maxContextDocs int, systemPrompt string, userPrompt func(query string, context string) (string, error)) Handler {
	return Handler{
		log:            log,
		embedder:       embedder,
		llm:            llm,
		queries:        queries,
		maxContextDocs: maxContextDocs,
		systemPrompt:   systemPrompt,
		userPrompt:     userPrompt,
	}
}

type Handler struct {
	log            *slog.Logger
	embedder       embeddings.Embedder
	llm            llms.Model
	queries        *db.Queries
	maxContextDocs int
	systemPrompt   string
	userPrompt     func(query string, context string) (string, error)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	partition, ok := auth.GetUser(r)
	if !ok {
		http.Error(w, "authentication not provided", http.StatusUnauthorized)
		return
	}

	var req models.QueryPostRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.log.Error("failed to decode body", slog.Any("error", err))
		respond.WithError(w, "failed to decode body", http.StatusBadRequest)
		return
	}

	var docs []db.DocumentSelectNearestResult

	if !req.NoContext {
		embedding, err := h.embedder.EmbedQuery(r.Context(), req.Text)
		if err != nil {
			h.log.Error("failed to embed query", slog.Any("error", err))
			respond.WithError(w, "failed to embed query", http.StatusInternalServerError)
			return
		}

		//TODO: Add metrics for query time. Use partition as a dimension.
		// Find the most similar documents.
		docs, err = h.queries.DocumentNearest(r.Context(), db.DocumentSelectNearestArgs{
			Partition: partition,
			Embedding: embedding,
			Limit:     h.maxContextDocs,
		})
		if err != nil {
			h.log.Error("failed to find nearest documents", slog.Any("error", err))
			respond.WithError(w, "failed to find nearest documents", http.StatusInternalServerError)
			return
		}
	}

	var sb strings.Builder
	for _, doc := range docs {
		sb.WriteString("Context from ")
		sb.WriteString(doc.Title)
		sb.WriteString(" - ")
		sb.WriteString(doc.URL)
		sb.WriteString("\n")
		sb.WriteString(doc.Text)
		sb.WriteString("\n")
	}
	prompt, err := h.userPrompt(req.Text, sb.String())
	if err != nil {
		h.log.Error("failed to generate prompt", slog.Any("error", err))
		respond.WithError(w, "failed to generate prompt", http.StatusInternalServerError)
		return
	}

	docIDs := make([]db.DocumentID, 0, len(docs))
	for i, doc := range docs {
		docIDs[i] = db.DocumentID{
			Partition: partition,
			URL:       doc.URL,
		}
	}
	h.log.Info("query context", slog.Any("docs", docIDs))

	f := func(ctx context.Context, chunk []byte) error {
		select {
		case <-ctx.Done():
			return nil
		default:
			if _, err := w.Write(chunk); err != nil {
				return err
			}
			if flusher, canFlush := w.(http.Flusher); canFlush {
				flusher.Flush()
			}
			return nil
		}
	}

	_, err = h.llm.GenerateContent(r.Context(), []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, h.systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}, llms.WithStreamingFunc(f))
	if err != nil {
		h.log.Error("failed to generate content", slog.Any("error", err))
		respond.WithError(w, "failed to generate content", http.StatusInternalServerError)
		return
	}
}
