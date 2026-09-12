package checker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/updu/updu/internal/models"
)

func TestHTTPChecker_HopTrace(t *testing.T) {
	checker := &HTTPChecker{}
	ctx := context.WithValue(context.Background(), AllowLocalhostKey, true)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok payload"))
	}))
	defer ts.Close()

	m := &models.Monitor{
		ID: "hop-test",
		Config: json.RawMessage(`{
			"url": "` + ts.URL + `",
			"method": "GET"
		}`),
		TimeoutS: 5,
	}

	res, err := checker.Check(ctx, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Status != models.StatusUp {
		t.Fatalf("expected up status, got %s", res.Status)
	}

	if len(res.Metadata) == 0 {
		t.Fatalf("expected metadata to be populated")
	}

	var meta struct {
		Trace *models.HopTrace `json:"trace"`
	}
	if err := json.Unmarshal(res.Metadata, &meta); err != nil {
		t.Fatalf("failed to unmarshal metadata: %v", err)
	}

	if meta.Trace == nil {
		t.Fatalf("expected trace in metadata")
	}

	if meta.Trace.ConnectedAddr == "" {
		t.Errorf("expected connected address in trace, got empty")
	}
}
