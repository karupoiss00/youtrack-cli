package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	appcli "github.com/urfave/cli/v3"

	"github.com/UsingCoding/youtrack-cli/internal/app"
)

func apiCommand(deps Dependencies) *appcli.Command {
	return &appcli.Command{
		Name: "api", Usage: "call the YouTrack REST API with current credentials", ArgsUsage: "<endpoint>",
		DisableSliceFlagSeparator: true,
		Flags: []appcli.Flag{
			&appcli.StringFlag{Name: "method", Value: http.MethodGet},
			&appcli.StringFlag{Name: "data"},
			&appcli.StringFlag{Name: "data-file"},
			&appcli.StringSliceFlag{Name: "header"},
		},
		Action: func(ctx context.Context, cmd *appcli.Command) error {
			args := cmd.Args().Slice()
			if err := requireArgs(args, 1, 1, "youtrack api <endpoint>"); err != nil {
				return err
			}
			if cmd.IsSet("data") && cmd.IsSet("data-file") {
				return app.Validationf("--data and --data-file are mutually exclusive")
			}
			var body []byte
			if cmd.IsSet("data") {
				body = []byte(cmd.String("data"))
			}
			if cmd.IsSet("data-file") {
				data, err := os.ReadFile(cmd.String("data-file"))
				if err != nil {
					return err
				}
				body = data
			}
			headers := http.Header{}
			for _, raw := range cmd.StringSlice("header") {
				key, value, ok := strings.Cut(raw, ":")
				if !ok {
					return app.Validationf("invalid --header %q: expected Key:Value", raw)
				}
				key = strings.TrimSpace(key)
				value = strings.TrimSpace(value)
				if strings.EqualFold(key, "Authorization") {
					return app.Validationf("Authorization header cannot be overridden")
				}
				headers.Add(key, value)
			}
			rt, err := buildRuntime(deps, cmd)
			if err != nil {
				return err
			}
			if rt.access.Restricted() {
				return app.Authf("raw API is disabled by local access restrictions; use structured commands")
			}
			resp, err := rt.client.DoRaw(ctx, cmd.String("method"), args[0], body, headers)
			if err != nil {
				return err
			}
			if globalBool(cmd, "plain") {
				_, err = deps.Out.Write(resp.Body)
				if err == nil && len(resp.Body) > 0 && resp.Body[len(resp.Body)-1] != '\n' {
					_, err = fmt.Fprintln(deps.Out)
				}
				return err
			}
			if globalBool(cmd, "json") {
				var v any
				if json.Unmarshal(resp.Body, &v) == nil {
					return rt.renderer.JSON(v)
				}
			}
			_, err = deps.Out.Write(resp.Body)
			if err == nil && len(resp.Body) > 0 && resp.Body[len(resp.Body)-1] != '\n' {
				_, err = fmt.Fprintln(deps.Out)
			}
			return err
		},
	}
}
