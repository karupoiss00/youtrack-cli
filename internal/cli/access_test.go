package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/config"
)

func TestRestrictedProfileDisablesRawAPIAndBrowserSearchBeforeHTTP(t *testing.T) {
	for _, profile := range []config.Profile{{DeniedProjects: []string{"SECRET"}}, {DeniedTags: []string{"private"}}} {
		for _, command := range [][]string{
			{"api", "/api/issues"},
			{"api", "/api/issues/2-1", "--method", "POST", "--data", `{}`},
			{"issue", "search", "all", "open", "--print-url"},
			{"saved-search", "open", "Mine", "--print-url"},
		} {
			t.Run(command[0]+"/"+command[1], func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
				defer server.Close()
				profile.URL = server.URL
				cfg := &mutableConfigStore{value: config.Config{Current: "work", Profiles: map[string]config.Profile{"work": profile}}}
				opener := &browserOpenerFake{}
				out := &bytes.Buffer{}
				root := browserRoot(out, Dependencies{Config: cfg, Credentials: cliCredentialsFake{}, BrowserOpener: opener, HTTPClient: server.Client()})
				args := append([]string{"youtrack"}, command...)
				args = append(args, "--url", server.URL, "--token", "secret", "--json")
				err := root.Run(context.Background(), args)
				require.Error(t, err)
				assert.Equal(t, app.ErrorAuth, app.KindOf(err))
				assert.Contains(t, err.Error(), "local access restrictions")
				assert.Zero(t, calls)
				assert.Empty(t, out.String())
				assert.Empty(t, opener.urls)
			})
		}
	}
}

func TestRestrictedProfileSelectionAndNestedFlagsProtectIssueContent(t *testing.T) {
	for _, selection := range []string{"current", "environment", "flag"} {
		t.Run(selection, func(t *testing.T) {
			t.Setenv("YOUTRACK_PROFILE", "")
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				assert.Equal(t, "/api/issues/2-1", r.URL.Path)
				assert.Equal(t, "id,project(id,name,shortName)", r.URL.Query().Get("fields"))
				_, _ = w.Write([]byte(`{"id":"2-1","project":{"id":"0-1","shortName":"SECRET"}}`))
			}))
			defer server.Close()
			cfg := &mutableConfigStore{value: config.Config{Current: "work", Profiles: map[string]config.Profile{
				"work":  {URL: server.URL, DeniedProjects: []string{"SECRET"}},
				"other": {URL: server.URL},
			}}}
			args := []string{"youtrack", "issue", "view", "2-1", "--url", server.URL, "--token", "secret", "--json"}
			if selection != "current" {
				cfg.value.Current = "other"
			}
			if selection == "environment" {
				t.Setenv("YOUTRACK_PROFILE", "work")
			}
			if selection == "flag" {
				t.Setenv("YOUTRACK_PROFILE", "other")
				args = append(args, "--profile", "work")
			}
			out := &bytes.Buffer{}
			err := browserRoot(out, Dependencies{Config: cfg, Credentials: cliCredentialsFake{}, HTTPClient: server.Client()}).Run(context.Background(), args)
			require.Error(t, err)
			assert.Equal(t, app.ErrorAuth, app.KindOf(err))
			assert.Equal(t, 1, requests)
			assert.Empty(t, out.String())
		})
	}
}

func TestConfigAccessKeysUseJSONArrays(t *testing.T) {
	store := &config.Store{Path: filepath.Join(t.TempDir(), "config.toml")}
	require.NoError(t, store.Set("profile.work.url", "https://youtrack.example"))
	out := &bytes.Buffer{}
	root := browserRoot(out, Dependencies{Config: store})
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "config", "set", "profile.work.denied_tags", `["private","tag,with,commas"]`}))
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "config", "get", "profile.work.denied_tags"}))
	assert.JSONEq(t, `["private","tag,with,commas"]`, out.String())
	out.Reset()
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "config", "list", "--json"}))
	assert.Contains(t, out.String(), `"DeniedTags"`)
	out.Reset()
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "config", "list"}))
	assert.Contains(t, out.String(), "denied_tags\t[\"private\",\"tag,with,commas\"]")
}

func TestAuthLoginPreservesAccessRestrictions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/users/me", r.URL.Path)
		_, _ = w.Write([]byte(`{"id":"u-1","login":"alice"}`))
	}))
	defer server.Close()
	want := config.Profile{URL: server.URL, DeniedProjects: []string{"SECRET"}, DeniedTags: []string{"private"}}
	cfg := &mutableConfigStore{value: config.Config{Profiles: map[string]config.Profile{"work": want}}}
	creds := &mutableCredentialStore{values: map[string]string{}}
	err := browserRoot(&bytes.Buffer{}, Dependencies{Config: cfg, Credentials: creds, HTTPClient: server.Client()}).Run(context.Background(), []string{"youtrack", "auth", "login", "--profile", "work", "--url", server.URL, "--token", "secret"})
	require.NoError(t, err)
	assert.Equal(t, want, cfg.value.Profiles["work"])
}
