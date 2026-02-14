package audit

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Gustik/shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testEvent() model.AuditEvent {
	return model.AuditEvent{
		Timestamp: 1234567890,
		Action:    "shorten",
		UserID:    "user-123",
		URL:       "https://example.com",
	}
}

func TestFileObserver_Notify(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "audit_*.log")
	require.NoError(t, err)
	defer f.Close()

	obs := NewFileObserver(f)
	event := testEvent()

	err = obs.Notify(event)
	require.NoError(t, err)

	content, err := os.ReadFile(f.Name())
	require.NoError(t, err)

	var got model.AuditEvent
	err = json.Unmarshal(content[:len(content)-1], &got) // убираем \n
	require.NoError(t, err)

	assert.Equal(t, event, got)
}

func TestFileObserver_Notify_MultipleEvents(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "audit_*.log")
	require.NoError(t, err)
	defer f.Close()

	obs := NewFileObserver(f)

	events := []model.AuditEvent{
		{Timestamp: 1, Action: "shorten", UserID: "u1", URL: "https://a.com"},
		{Timestamp: 2, Action: "follow", UserID: "u2", URL: "https://b.com"},
		{Timestamp: 3, Action: "shorten", UserID: "u3", URL: "https://c.com"},
	}

	for _, e := range events {
		require.NoError(t, obs.Notify(e))
	}

	content, err := os.ReadFile(f.Name())
	require.NoError(t, err)

	decoder := json.NewDecoder(strings.NewReader(string(content)))
	var got []model.AuditEvent
	for decoder.More() {
		var e model.AuditEvent
		require.NoError(t, decoder.Decode(&e))
		got = append(got, e)
	}

	assert.Equal(t, events, got)
}

func TestFileObserver_GetID(t *testing.T) {
	obs := &FileObserver{}
	assert.Equal(t, "file", obs.GetID())
}

func TestHTTPObserver_Notify(t *testing.T) {
	var receivedBody []byte
	var receivedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	obs := NewHTTPObserver(server.URL)
	event := testEvent()

	err := obs.Notify(event)
	require.NoError(t, err)

	assert.Equal(t, "application/json", receivedContentType)

	var got model.AuditEvent
	require.NoError(t, json.Unmarshal(receivedBody, &got))
	assert.Equal(t, event, got)
}

func TestHTTPObserver_Notify_ServerUnavailable(t *testing.T) {
	obs := NewHTTPObserver("http://localhost:1")

	err := obs.Notify(testEvent())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send audit event")
}

func TestHTTPObserver_GetID(t *testing.T) {
	obs := &HTTPObserver{}
	assert.Equal(t, "http", obs.GetID())
}
