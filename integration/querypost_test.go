package integration

import (
	"bytes"
	"context"
	"testing"

	"github.com/a-h/ragserver/client"
	querypost "github.com/a-h/ragserver/handlers/query/post"
	"github.com/a-h/ragserver/models"
)

func TestQueryPost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	buf := new(bytes.Buffer)
	f := func(ctx context.Context, chunk []byte) (err error) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_, err = buf.Write(chunk)
		return err
	}
	c := client.New("http://localhost:9020", "test-api-key-no-llm")
	err := c.QueryPost(context.Background(), models.QueryPostRequest{
		Text:      "This is a test query.",
		NoContext: false,
	}, f)
	if err != nil {
		t.Fatalf("failed to post query: %v", err)
	}
	actual := buf.String()
	if actual != querypost.TestMessage {
		t.Fatalf("expected %q, got %q", querypost.TestMessage, actual)
	}
}
