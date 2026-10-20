package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
)

func TestStoreRoundTripWithBurntSushiTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "youtrack", "config.toml")
	store, err := NewStore(path)
	require.NoError(t, err)

	want := Config{Current: "company", Profiles: map[string]Profile{"company": {URL: "https://example.youtrack.cloud"}}}
	require.NoError(t, store.Save(want))
	got, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, want, got)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

type testCreds map[string]string

func (c testCreds) Get(profile string) (string, error) { return c[profile], nil }

type failingCreds struct{ err error }

func (c failingCreds) Get(string) (string, error) { return "", c.err }

func TestResolveDoesNotReuseStoredTokenForURLOverride(t *testing.T) {
	t.Setenv("YOUTRACK_URL", "")
	t.Setenv("YOUTRACK_TOKEN", "")
	t.Setenv("YOUTRACK_PROFILE", "")
	cfg := Config{Current: "company", Profiles: map[string]Profile{"company": {URL: "https://old.example"}}}

	_, err := Resolve(cfg, testCreds{"company": "stored-secret"}, RuntimeInput{URL: "https://new.example"})
	require.Error(t, err)

	got, err := Resolve(cfg, testCreds{"company": "stored-secret"}, RuntimeInput{URL: "https://new.example", Token: "explicit"})
	require.NoError(t, err)
	assert.Equal(t, "https://new.example", got.URL)
	assert.Equal(t, "explicit", got.Token)
}

func TestStoreUsesXDGAndHandlesMissingMalformedAndNilProfiles(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	path, err := DefaultPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "youtrack", "config.toml"), path)
	store, err := NewStore("")
	require.NoError(t, err)
	assert.Equal(t, path, store.Path)

	cfg, err := store.Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.Current)
	assert.NotNil(t, cfg.Profiles)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("not = [valid"), 0o600))
	_, err = store.Load()
	require.Error(t, err)
	require.NoError(t, os.WriteFile(path, []byte("current = \"\"\n"), 0o600))
	cfg, err = store.Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg.Profiles)
}

func TestStoreSetAndGetSupportedKeys(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "config.toml"))
	require.NoError(t, err)
	require.NoError(t, store.Set("profile.company.url", "https://company.example"))
	current, err := store.Get("current")
	require.NoError(t, err)
	assert.Equal(t, "company", current)
	url, err := store.Get("profile.company.url")
	require.NoError(t, err)
	assert.Equal(t, "https://company.example", url)
	require.NoError(t, store.Set("profile.personal.url", "https://personal.example"))
	require.NoError(t, store.Set("current", "personal"))
	current, err = store.Get("current")
	require.NoError(t, err)
	assert.Equal(t, "personal", current)
	for _, key := range []string{"current.missing", "profile.missing.url"} {
		_, err = store.Get(key)
		require.Error(t, err)
	}
	for _, key := range []string{"unsupported", "current"} {
		err = store.Set(key, "missing")
		require.Error(t, err)
	}
}

func TestResolvePrecedenceAndAuthFailures(t *testing.T) {
	t.Setenv("YOUTRACK_PROFILE", "")
	t.Setenv("YOUTRACK_URL", "")
	t.Setenv("YOUTRACK_TOKEN", "")
	cfg := Config{Current: "current", Profiles: map[string]Profile{
		"current": {URL: "https://current.example"},
		"chosen":  {URL: "https://chosen.example"},
		"empty":   {},
	}}
	t.Run("explicit flags override configured profile", func(t *testing.T) {
		got, err := Resolve(cfg, testCreds{}, RuntimeInput{Profile: "chosen", URL: "https://override.example", Token: "explicit"})
		require.NoError(t, err)
		assert.Equal(t, Runtime{Profile: "chosen", URL: "https://override.example", Token: "explicit"}, got)
	})
	t.Run("environment selects profile and credentials", func(t *testing.T) {
		t.Setenv("YOUTRACK_PROFILE", "chosen")
		t.Setenv("YOUTRACK_TOKEN", "environment")
		got, err := Resolve(cfg, testCreds{}, RuntimeInput{})
		require.NoError(t, err)
		assert.Equal(t, Runtime{Profile: "chosen", URL: "https://chosen.example", Token: "environment"}, got)
	})
	t.Run("explicit token overrides stored token", func(t *testing.T) {
		got, err := Resolve(cfg, testCreds{"current": "stored"}, RuntimeInput{Token: "explicit"})
		require.NoError(t, err)
		assert.Equal(t, "explicit", got.Token)
	})
	t.Run("stored token resolves current profile", func(t *testing.T) {
		got, err := Resolve(cfg, testCreds{"current": "stored"}, RuntimeInput{})
		require.NoError(t, err)
		assert.Equal(t, Runtime{Profile: "current", URL: "https://current.example", Token: "stored"}, got)
	})
	for _, tt := range []struct {
		name string
		cfg  Config
		in   RuntimeInput
		cred CredentialReader
	}{
		{name: "no profile", cfg: Config{Profiles: map[string]Profile{}}, cred: testCreds{}},
		{name: "missing profile", cfg: cfg, in: RuntimeInput{Profile: "missing"}, cred: testCreds{}},
		{name: "profile without URL", cfg: cfg, in: RuntimeInput{Profile: "empty"}, cred: testCreds{}},
		{name: "missing token", cfg: cfg, cred: testCreds{}},
		{name: "credential error", cfg: cfg, cred: failingCreds{err: os.ErrPermission}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("YOUTRACK_PROFILE", "")
			t.Setenv("YOUTRACK_URL", "")
			t.Setenv("YOUTRACK_TOKEN", "")
			_, err := Resolve(tt.cfg, tt.cred, tt.in)
			require.Error(t, err)
			if tt.name != "credential error" {
				assert.Equal(t, app.ErrorAuth, app.KindOf(err))
			}
		})
	}
}

func TestStoreNormalizesNilProfilesAndPropagatesFilesystemFailures(t *testing.T) {
	t.Run("save normalizes nil profiles", func(t *testing.T) {
		store, err := NewStore(filepath.Join(t.TempDir(), "config.toml"))
		require.NoError(t, err)
		require.NoError(t, store.Save(Config{}))
		cfg, err := store.Load()
		require.NoError(t, err)
		assert.NotNil(t, cfg.Profiles)
	})
	t.Run("load normalizes decoded nil profiles", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.toml")
		require.NoError(t, os.WriteFile(path, []byte("current = \"company\"\n"), 0o600))
		cfg, err := (&Store{Path: path}).Load()
		require.NoError(t, err)
		assert.NotNil(t, cfg.Profiles)
	})
	t.Run("load get set and save return filesystem errors", func(t *testing.T) {
		parent := filepath.Join(t.TempDir(), "not-a-directory")
		require.NoError(t, os.WriteFile(parent, []byte("file"), 0o600))
		store := &Store{Path: filepath.Join(parent, "config.toml")}
		_, err := store.Load()
		require.Error(t, err)
		_, err = store.Get("current")
		require.Error(t, err)
		err = store.Set("profile.company.url", "https://company.example")
		require.Error(t, err)
		err = store.Save(Config{})
		require.Error(t, err)
	})
}

func TestDefaultPathFallsBackToHomeConfigurationDirectory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	path, err := DefaultPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".config", "youtrack", "config.toml"), path)
	store, err := NewStore("")
	require.NoError(t, err)
	assert.Equal(t, path, store.Path)
}

type countingCreds struct {
	calls int
	err   error
}

func (c *countingCreds) Get(string) (string, error) {
	c.calls++
	return "", c.err
}

func TestResolveServicePrecedenceDoesNotReadCredentials(t *testing.T) {
	t.Setenv("YOUTRACK_PROFILE", "")
	t.Setenv("YOUTRACK_URL", "")
	cfg := Config{Current: "current", Profiles: map[string]Profile{
		"current": {URL: "https://current.example"},
		"chosen":  {URL: "https://chosen.example"},
	}}
	creds := &countingCreds{err: os.ErrPermission}

	for _, tc := range []struct {
		name    string
		profile string
		url     string
		envURL  string
		want    Service
	}{
		{name: "flag URL", profile: "chosen", url: "https://flag.example", want: Service{Profile: "chosen", URL: "https://flag.example"}},
		{name: "environment URL", profile: "chosen", envURL: "https://environment.example", want: Service{Profile: "chosen", URL: "https://environment.example"}},
		{name: "selected profile", profile: "chosen", want: Service{Profile: "chosen", URL: "https://chosen.example"}},
		{name: "current profile", want: Service{Profile: "current", URL: "https://current.example"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("YOUTRACK_URL", tc.envURL)
			got, err := ResolveService(cfg, ServiceInput{Profile: tc.profile, URL: tc.url})
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
			assert.Zero(t, creds.calls)
		})
	}
}

func TestResolveServiceValidatesSelectedProfile(t *testing.T) {
	t.Setenv("YOUTRACK_PROFILE", "")
	t.Setenv("YOUTRACK_URL", "")
	cfg := Config{Profiles: map[string]Profile{"blank": {URL: " "}}}
	for _, in := range []ServiceInput{{}, {Profile: "missing"}, {Profile: "blank"}} {
		_, err := ResolveService(cfg, in)
		require.Error(t, err)
		assert.Equal(t, app.ErrorAuth, app.KindOf(err))
	}
}
