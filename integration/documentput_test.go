package integration

import (
	"context"
	"testing"

	"github.com/a-h/ragserver/client"
	"github.com/a-h/ragserver/models"
)

func TestDocumentPut(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	c := client.New("http://localhost:9020", "test-api-key-no-llm")
	_, err := c.DocumentsPut(context.Background(), models.DocumentsPostRequest{
		Document: models.Document{
			URL:     "/test",
			Title:   "A test document",
			Text:    "This is a test document. It is used to test the document post endpoint.",
			Summary: "Summary of the test document.",
		},
	})
	if err != nil {
		t.Fatalf("failed to put document: %v", err)
	}
}
