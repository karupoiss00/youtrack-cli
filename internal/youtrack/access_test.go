package youtrack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func TestIssueAccessPaginatesTagsPastServerDefault(t *testing.T) {
	var skips []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		switch r.URL.Path {
		case "/youtrack/api/issues/2-1":
			assert.Equal(t, "id,project(id,name,shortName)", r.URL.Query().Get("fields"))
			_, _ = w.Write([]byte(`{"id":"2-1","project":{"id":"0-1","name":"Tools","shortName":"TT"}}`))
		case "/youtrack/api/issues/2-1/tags":
			assert.Equal(t, tagFields, r.URL.Query().Get("fields"))
			assert.Equal(t, "100", r.URL.Query().Get("$top"))
			skip := r.URL.Query().Get("$skip")
			skips = append(skips, skip)
			items := []map[string]string{}
			switch skip {
			case "0":
				for i := range 42 {
					items = append(items, map[string]string{"id": fmt.Sprint(i), "name": "public"})
				}
			case "42":
				items = append(items, map[string]string{"id": "private-id", "name": "private"})
			case "43":
			default:
				t.Errorf("unexpected offset %s", skip)
			}
			require.NoError(t, json.NewEncoder(w).Encode(items))
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL + "/youtrack", HTTPClient: server.Client()})
	require.NoError(t, err)
	got, err := client.GetIssueAccess(context.Background(), "2-1", true)
	require.NoError(t, err)
	assert.Equal(t, domain.Project{ID: "0-1", Name: "Tools", ShortName: "TT"}, got.Project)
	require.Len(t, got.Tags, 43)
	assert.Equal(t, domain.Tag{ID: "private-id", Name: "private"}, got.Tags[42])
	assert.Equal(t, []string{"0", "42", "43"}, skips)
	skips = nil
	_, err = client.GetIssueAccess(context.Background(), "2-1", false)
	require.NoError(t, err)
	assert.Empty(t, skips)
}

func TestIssueAccessPropagatesMetadataAndTagErrors(t *testing.T) {
	for _, failTags := range []bool{false, true} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if failTags && r.URL.Path == "/api/issues/2-1" {
				_, _ = w.Write([]byte(`{"project":{"id":"0-1"}}`))
				return
			}
			http.Error(w, `{"error_description":"forbidden"}`, http.StatusForbidden)
		}))
		client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
		require.NoError(t, err)
		_, err = client.GetIssueAccess(context.Background(), "2-1", true)
		require.Error(t, err)
		assert.Equal(t, app.ErrorAuth, app.KindOf(err))
		server.Close()
	}
}

func TestIssueAccessRejectsIncompleteMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"2-1"}`))
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)
	_, err = client.GetIssueAccess(context.Background(), "2-1", false)
	require.ErrorContains(t, err, "no project ID")
}
