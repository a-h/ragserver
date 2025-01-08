package post

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/a-h/ragserver/auth"
	"github.com/a-h/ragserver/db"
	"github.com/a-h/ragserver/models"
	"github.com/a-h/respond"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
)

func New(log *slog.Logger, embedder embeddings.Embedder, llm llms.Model, queries *db.Queries, maxContextDocs int) Handler {
	return Handler{
		log:            log,
		embedder:       embedder,
		queries:        queries,
		maxContextDocs: maxContextDocs,
	}
}

type Handler struct {
	log            *slog.Logger
	embedder       embeddings.Embedder
	queries        *db.Queries
	maxContextDocs int
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUser(r)
	if !ok {
		http.Error(w, "authentication not provided", http.StatusUnauthorized)
		return
	}

	var req models.ContextPostRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.log.Error("failed to decode body", slog.Any("error", err))
		respond.WithError(w, "failed to decode body", http.StatusBadRequest)
		return
	}

	var docs []db.DocumentSelectNearestResult

	// If this is a test API key, don't use the LLM.
	if req.Text != "" && user != "test-user-no-llm" {
		embedding, err := h.embedder.EmbedQuery(r.Context(), req.Text)
		if err != nil {
			h.log.Error("failed to embed query", slog.Any("error", err))
			respond.WithError(w, "failed to embed query", http.StatusInternalServerError)
			return
		}

		//TODO: Add metrics for query time. Use partition as a dimension.
		// Find the most similar documents.
		docs, err = h.queries.DocumentNearest(r.Context(), db.DocumentSelectNearestArgs{
			Partition: user,
			Embedding: embedding,
			Limit:     h.maxContextDocs,
		})
		if err != nil {
			h.log.Error("failed to find nearest documents", slog.Any("error", err))
			respond.WithError(w, "failed to find nearest documents", http.StatusInternalServerError)
			return
		}
	}

	var qpr models.ContextPostResponse
	for _, doc := range docs {
		qpr.Results = append(qpr.Results, models.ContextDocument{
			Text:      doc.Text,
			Embedding: doc.Embedding,
			Distance:  doc.Distance,
			URL:       doc.URL,
			Title:     doc.Title,
			Summary:   doc.Summary,
		})
	}

	respond.WithJSON(w, qpr, http.StatusOK)
}
