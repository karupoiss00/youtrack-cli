package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreAccessRestrictionsRoundTripAndKeys(t *testing.T) {
	store := &Store{Path: filepath.Join(t.TempDir(), "config.toml")}
	require.NoError(t, store.Set("profile.work.url", "https://youtrack.example"))
	for _, key := range []string{"denied_projects", "denied_tags"} {
		full := "profile.work." + key
		got, err := store.Get(full)
		require.NoError(t, err)
		assert.Equal(t, "[]", got)
		require.NoError(t, store.Set(full, `["Secret Project","tag,with,commas","quotes\"inside"]`))
		got, err = store.Get(full)
		require.NoError(t, err)
		assert.JSONEq(t, `["Secret Project","tag,with,commas","quotes\"inside"]`, got)
		cfg, err := store.Load()
		require.NoError(t, err)
		assert.True(t, cfg.Profiles["work"].AccessPolicy().Restricted())
		for _, invalid := range []string{"SECRET", "null", `"SECRET"`, `[1]`, `[null]`, `[""]`, `["  "]`, `[] trailing`} {
			require.Error(t, store.Set(full, invalid), invalid)
			unchanged, err := store.Get(full)
			require.NoError(t, err)
			assert.Equal(t, got, unchanged)
		}
		require.NoError(t, store.Set(full, `[]`))
		got, err = store.Get(full)
		require.NoError(t, err)
		assert.Equal(t, "[]", got)
		_, err = store.Get("profile.missing." + key)
		require.Error(t, err)
		require.Error(t, store.Set("profile.missing."+key, `[]`))
	}
}

func TestStoreRejectsMalformedAccessConfiguration(t *testing.T) {
	for _, body := range []string{
		"[profiles.work]\ndenied_tags = [\"\"]\n",
		"[profiles.work]\ndenied_projects = [\" \" ]\n",
		"[profiles.work]\ndenied_projects = \"SECRET\"\n",
	} {
		path := filepath.Join(t.TempDir(), "config.toml")
		require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
		_, err := (&Store{Path: path}).Load()
		require.Error(t, err)
	}
	store := &Store{Path: filepath.Join(t.TempDir(), "config.toml")}
	require.Error(t, store.Save(Config{Profiles: map[string]Profile{"work": {DeniedTags: []string{""}}}}))
	_, err := os.Stat(store.Path)
	assert.True(t, os.IsNotExist(err))
}
