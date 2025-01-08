package post

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/a-h/ragserver/auth"
	"github.com/a-h/ragserver/models"
	"github.com/a-h/respond"
	"github.com/tmc/langchaingo/llms"
)

func New(log *slog.Logger, llm llms.Model) Handler {
	return Handler{
		log: log,
		llm: llm,
	}
}

type Handler struct {
	log *slog.Logger
	llm llms.Model
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUser(r)
	if !ok {
		http.Error(w, "authentication not provided", http.StatusUnauthorized)
		return
	}

	var req models.ChatPostRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.log.Error("failed to decode body", slog.Any("error", err))
		respond.WithError(w, "failed to decode body", http.StatusBadRequest)
		return
	}

	// If this is a test API key, don't use the LLM.
	if user == "test-user-no-llm" {
		writeTestMessage(w)
		return
	}

	var msgs []llms.MessageContent
	for _, m := range req.Messages {
		msgs = append(msgs, llms.TextParts(llms.ChatMessageType(m.Type), m.Content))
	}

	h.log.Info("generating content", slog.Any("messages", msgs))

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

	_, err = h.llm.GenerateContent(r.Context(), msgs, llms.WithStreamingFunc(f))
	if err != nil {
		h.log.Error("failed to generate content", slog.Any("error", err))
		respond.WithError(w, "failed to generate content", http.StatusInternalServerError)
		return
	}
}

const TestMessage = `Hello!

I'm a test message.

I'm here to help you test your integration with the API.

If you can see me, then your integration is working!`

func writeTestMessage(w http.ResponseWriter) (err error) {
	for chunk := range slices.Chunk([]rune(TestMessage), 4) {
		if _, err := io.WriteString(w, string(chunk)); err != nil {
			return err
		}
		if flusher, canFlush := w.(http.Flusher); canFlush {
			flusher.Flush()
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}
