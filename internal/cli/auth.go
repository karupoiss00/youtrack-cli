package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	appcli "github.com/urfave/cli/v3"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/config"
	"github.com/UsingCoding/youtrack-cli/internal/youtrack"
)

func authCommand(deps Dependencies) *appcli.Command {
	return &appcli.Command{Name: "auth", Usage: "manage authentication", Commands: []*appcli.Command{
		{Name: "login", Usage: "log in to a YouTrack server", Action: func(ctx context.Context, cmd *appcli.Command) error { return authLogin(ctx, deps, cmd) }},
		{Name: "logout", Usage: "remove stored credentials", Action: func(ctx context.Context, cmd *appcli.Command) error { return authLogout(deps, cmd) }},
		{Name: "status", Usage: "show current authentication status", Action: func(ctx context.Context, cmd *appcli.Command) error { return authStatus(ctx, deps, cmd) }},
	}}
}

func authLogin(ctx context.Context, deps Dependencies, cmd *appcli.Command) error {
	cfg, err := deps.Config.Load()
	if err != nil {
		return err
	}
	profile := globalString(cmd, "profile")
	if profile == "" {
		profile = os.Getenv("YOUTRACK_PROFILE")
	}
	if profile == "" {
		profile = "default"
	}

	serverURL := globalString(cmd, "url")
	if serverURL == "" {
		serverURL = os.Getenv("YOUTRACK_URL")
	}
	if serverURL == "" {
		if p, ok := cfg.Profiles[profile]; ok {
			serverURL = p.URL
		}
	}
	if serverURL == "" {
		serverURL, err = readLine(deps.In, deps.Err, "YouTrack URL: ")
		if err != nil {
			return err
		}
	}
	if strings.TrimSpace(serverURL) == "" {
		return app.Validationf("YouTrack URL cannot be empty")
	}

	token := globalString(cmd, "token")
	if token == "" {
		token = os.Getenv("YOUTRACK_TOKEN")
	}
	if token == "" {
		token, err = readSecret(deps.In, deps.Err, "Token: ")
		if err != nil {
			return err
		}
	}
	if token == "" {
		return app.Validationf("token cannot be empty")
	}

	client, err := youtrack.NewClient(youtrack.Options{BaseURL: serverURL, Token: token, HTTPClient: deps.HTTPClient, Timeout: 30 * time.Second, UserAgent: "youtrack-cli/" + deps.Version})
	if err != nil {
		return err
	}
	me, err := client.Me(ctx)
	if err != nil {
		return err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]config.Profile{}
	}
	p := cfg.Profiles[profile]
	p.URL = serverURL
	cfg.Profiles[profile] = p
	cfg.Current = profile
	if err := deps.Config.Save(cfg); err != nil {
		return err
	}
	if err := deps.Credentials.Set(profile, token); err != nil {
		return err
	}
	_, err = fmt.Fprintf(deps.Out, "Logged in to %s as %s\n", serverURL, me.Login)
	return err
}

func authLogout(deps Dependencies, cmd *appcli.Command) error {
	cfg, err := deps.Config.Load()
	if err != nil {
		return err
	}
	profile := globalString(cmd, "profile")
	if profile == "" {
		profile = os.Getenv("YOUTRACK_PROFILE")
	}
	if profile == "" {
		profile = cfg.Current
	}
	if profile == "" {
		return app.Authf("no YouTrack profile is selected")
	}
	if err := deps.Credentials.Delete(profile); err != nil {
		return err
	}
	_, err = fmt.Fprintf(deps.Out, "Logged out from profile %s\n", profile)
	return err
}

func authStatus(ctx context.Context, deps Dependencies, cmd *appcli.Command) error {
	rt, err := buildRuntime(deps, cmd)
	if err != nil {
		return err
	}
	me, err := rt.client.Me(ctx)
	if err != nil {
		return err
	}
	if globalBool(cmd, "json") {
		return rt.renderer.JSON(map[string]any{"profile": rt.profile, "url": rt.url, "user": map[string]any{"id": me.ID, "login": me.Login, "name": me.FullName}})
	}
	if globalBool(cmd, "plain") {
		_, err = fmt.Fprintln(deps.Out, me.Login)
		return err
	}
	_, err = fmt.Fprintf(deps.Out, "Logged in to %s as %s (profile %s)\n", rt.url, me.Login, rt.profile)
	return err
}
