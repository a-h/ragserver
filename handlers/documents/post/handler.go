package post

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"sync"

	"github.com/a-h/ragserver/auth"
	"github.com/a-h/ragserver/db"
	"github.com/a-h/ragserver/models"
	"github.com/a-h/respond"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/textsplitter"
)

func New(log *slog.Logger, embedder embeddings.Embedder, queries *db.Queries) Handler {
	splitter := textsplitter.NewMarkdownTextSplitter()

	return Handler{
		log:      log,
		splitter: splitter,
		embedder: embedder,
		queries:  queries,
	}
}

type Handler struct {
	log      *slog.Logger
	splitter textsplitter.TextSplitter
	embedder embeddings.Embedder
	queries  *db.Queries
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	partition, ok := auth.GetUser(r)
	if !ok {
		http.Error(w, "authentication not provided", http.StatusUnauthorized)
		return
	}

	var req models.DocumentsPostRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.log.Error("failed to decode body", slog.Any("error", err))
		respond.WithError(w, "failed to decode body", http.StatusBadRequest)
		return
	}

	texts, err := h.split(req.Document)
	if err != nil {
		h.log.Error("failed to split text", slog.Any("error", err))
		respond.WithError(w, "failed to split text", http.StatusInternalServerError)
		return
	}

	//TODO: Add metrics for text count, text length, and embedding time. Use partition as a dimension.
	embeddings, err := h.embedder.EmbedDocuments(r.Context(), texts)
	if err != nil {
		h.log.Error("failed to embed documents", slog.Any("error", err))
		respond.WithError(w, "failed to embed documents", http.StatusInternalServerError)
		return
	}

	if len(texts) != len(embeddings) {
		h.log.Error("length mismatch", slog.Int("texts", len(texts)), slog.Int("embeddings", len(embeddings)))
		respond.WithError(w, "split/embedding failed", http.StatusInternalServerError)
		return
	}

	chunks := make([]db.Chunk, len(texts))
	for i := 0; i < len(texts); i++ {
		chunks[i] = db.Chunk{
			Text:      texts[i],
			Embedding: embeddings[i],
		}
	}

	var resp models.DocumentsPostResponse
	resp.ID, err = h.queries.DocumentPut(r.Context(), db.DocumentPutArgs{
		Document: db.Document{
			DocumentID: db.DocumentID{
				Partition: partition,
				URL:       req.Document.URL,
			},
			Title:   req.Document.Title,
			Text:    req.Document.Text,
			Summary: req.Document.Summary,
		},
		Chunks: chunks,
	})
	if err != nil {
		h.log.Error("document put failed", slog.Any("error", err))
		respond.WithError(w, "document put failed", http.StatusInternalServerError)
		return
	}

	respond.WithJSON(w, resp, http.StatusOK)
}

func (h *Handler) split(d models.Document) ([]string, error) {
	inputs := []string{d.Title, d.Text, d.Summary}
	outputs := make([][]string, len(inputs))
	errs := make([]error, len(inputs))
	var wg sync.WaitGroup
	wg.Add(len(inputs))
	for i := range inputs {
		go func(i int) {
			defer wg.Done()
			outputs[i], errs[i] = h.splitter.SplitText(inputs[i])
		}(i)
	}
	wg.Wait()
	return slices.Concat(outputs...), errors.Join(errs...)
}
