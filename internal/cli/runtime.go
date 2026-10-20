package cli

import (
	"time"

	appcli "github.com/urfave/cli/v3"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/config"
	"github.com/UsingCoding/youtrack-cli/internal/output"
	"github.com/UsingCoding/youtrack-cli/internal/youtrack"
)

type runtime struct {
	access   app.AccessPolicy
	client   *youtrack.Client
	service  *app.Service
	renderer *output.Renderer
	profile  string
	url      string
}

type urlRuntime struct {
	renderer *output.Renderer
	url      string
}

func buildRuntime(deps Dependencies, cmd *appcli.Command) (*runtime, error) {
	cfg, err := deps.Config.Load()
	if err != nil {
		return nil, err
	}
	resolved, err := config.Resolve(cfg, deps.Credentials, config.RuntimeInput{
		Profile: globalString(cmd, "profile"), URL: globalString(cmd, "url"), Token: globalString(cmd, "token"),
	})
	if err != nil {
		return nil, err
	}
	timeout := globalDuration(cmd, "timeout")
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client, err := youtrack.NewClient(youtrack.Options{
		BaseURL: resolved.URL, Token: resolved.Token, HTTPClient: deps.HTTPClient,
		Timeout: timeout, UserAgent: "youtrack-cli/" + deps.Version,
	})
	if err != nil {
		return nil, err
	}
	renderer, err := output.New(deps.Out, globalBool(cmd, "json"), globalBool(cmd, "plain"))
	if err != nil {
		return nil, err
	}
	return &runtime{
		access:   cfg.Profiles[resolved.Profile].AccessPolicy(),
		client:   client,
		service:  app.NewService(client, client, client, client, client, client, client, client, client, client, client, client, client, client, client, client).WithAccessPolicy(cfg.Profiles[resolved.Profile].AccessPolicy(), client),
		renderer: renderer, profile: resolved.Profile, url: resolved.URL,
	}, nil
}

func buildURLRuntime(deps Dependencies, cmd *appcli.Command) (*urlRuntime, error) {
	cfg, err := deps.Config.Load()
	if err != nil {
		return nil, err
	}
	service, err := config.ResolveService(cfg, config.ServiceInput{
		Profile: globalString(cmd, "profile"),
		URL:     globalString(cmd, "url"),
	})
	if err != nil {
		return nil, err
	}
	if cfg.Profiles[service.Profile].AccessPolicy().Restricted() {
		return nil, app.Authf("browser search is disabled by local access restrictions; use issue search without open")
	}
	renderer, err := output.New(deps.Out, globalBool(cmd, "json"), globalBool(cmd, "plain"))
	if err != nil {
		return nil, err
	}
	return &urlRuntime{renderer: renderer, url: service.URL}, nil
}
