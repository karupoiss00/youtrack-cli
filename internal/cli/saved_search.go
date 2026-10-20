package cli

import (
	"context"
	"strings"

	appcli "github.com/urfave/cli/v3"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/browser"
)

func savedSearchCommand(deps Dependencies) *appcli.Command {
	return &appcli.Command{Name: "saved-search", Usage: "view saved searches", Commands: []*appcli.Command{
		savedSearchViewCommand(deps), savedSearchOpenCommand(deps),
	}}
}

func savedSearchViewCommand(deps Dependencies) *appcli.Command {
	return &appcli.Command{
		Name: "view", Usage: "view a saved search", ArgsUsage: "<saved-search>", Flags: paginationFlags(),
		Action: func(ctx context.Context, cmd *appcli.Command) error {
			args := cmd.Args().Slice()
			if err := requireArgs(args, 1, 1, "youtrack saved-search view <saved-search> [--limit <n>] [--offset <n>] [--all]"); err != nil {
				return err
			}
			if strings.TrimSpace(args[0]) == "" {
				return app.Validationf("saved search reference must not be blank")
			}
			request, err := pageRequest(cmd)
			if err != nil {
				return err
			}
			rt, err := buildRuntime(deps, cmd)
			if err != nil {
				return err
			}
			view, err := rt.service.ViewSavedSearch(ctx, args[0], request)
			if err != nil {
				return err
			}
			return rt.renderer.SavedSearch(view.Search, view.Issues)
		},
	}
}

func savedSearchOpenCommand(deps Dependencies) *appcli.Command {
	return &appcli.Command{
		Name: "open", Usage: "open a saved search in a browser", ArgsUsage: "<saved-search>", Flags: []appcli.Flag{
			&appcli.BoolFlag{Name: "print-url", Usage: "print the destination without opening a browser"},
		},
		Action: func(ctx context.Context, cmd *appcli.Command) error {
			args := cmd.Args().Slice()
			if err := requireArgs(args, 1, 1, "youtrack saved-search open <saved-search>"); err != nil {
				return err
			}
			if strings.TrimSpace(args[0]) == "" {
				return app.Validationf("saved search reference must not be blank")
			}
			rt, err := buildRuntime(deps, cmd)
			if err != nil {
				return err
			}
			if rt.access.Restricted() {
				return app.Authf("browser search is disabled by local access restrictions; use saved-search view")
			}
			search, err := rt.service.ResolveSavedSearch(ctx, args[0])
			if err != nil {
				return err
			}
			url, err := browser.BuildSearchURL(rt.url, search.Query)
			if err != nil {
				return err
			}
			return openURL(deps, rt.renderer, url, cmd.Bool("print-url"))
		},
	}
}
