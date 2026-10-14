# youtrack-cli

A Go CLI for JetBrains YouTrack, inspired by the entity-oriented structure and agent-friendly workflows of [`teamcity-cli`](https://github.com/jetbrains/teamcity-cli).

Inspect, search, create, and update issues from your terminal, with structured output for scripts and coding agents.

## Features

| Feature | Status | Supported operations |
| --- | --- | --- |
| Authentication | Supported | Login, logout, and status with named profiles |
| Issue inspection and search | Supported | View issues, run YouTrack queries, and paginate results |
| Issue creation and editing | Supported | Create issues; update summary, description, custom fields, and tags |
| Custom fields | Supported | List, read, set, and clear values resolved against project metadata |
| Board membership | Supported | Read and replace memberships through the synthetic `Board` field |
| Tags and project moves | Supported | List, add, and remove issue tags; move issues between projects |
| Saved searches | Supported | Run visible saved queries (read-only) |
| Comments | Supported | List, add, edit, and remove comments |
| Time tracking | Supported | Discover work-item types; list, view, add, edit, and remove work items |
| Issue relationships | Supported | Discover link types; list, add, and remove links, including subtask relationships |
| Browser handoff | Supported | Open issues, queries, and saved searches, or print their URLs |
| Raw REST API | Supported | Call relative `/api` endpoints not yet modeled by dedicated commands |
| Structured output | Supported | Human-readable, stable JSON, and shell-friendly plain output |
| Agent skill | Supported | Install and update the embedded coding-agent skill |

Implementation specifications: [`spec/mvp1`](spec/mvp1/00-overview.md), [`spec/mvp2`](spec/mvp2/00-overview.md), and [`spec/mvp3`](spec/mvp3/00-overview.md).

## Installation

With [mise installed and activated](https://mise.jdx.dev/getting-started.html), install the latest release globally using its [GitHub backend](https://mise.jdx.dev/dev-tools/backends/github.html):

```bash
mise use -g github:UsingCoding/youtrack-cli@latest
youtrack version
```

Omit `-g` to manage the CLI in the current project's `mise.toml` instead.

## Quick start

Install the CLI, then log in with your YouTrack server URL and permanent token:

```bash
mise use -g github:UsingCoding/youtrack-cli@latest
youtrack auth login
youtrack auth status
```

Replace `APP` and `APP-123` below with your project short name and an existing issue ID:

```bash
youtrack issue search 'project: APP #Unresolved' --limit 20
youtrack issue view APP-123
youtrack issue view APP-123 --json
```

## Install for development

```bash
mise install
mise run build
./bin/youtrack version
```

Or:

```bash
go install ./cmd/youtrack
```

## Authentication

```bash
youtrack auth login
```

You can also provide:

```text
YOUTRACK_PROFILE
YOUTRACK_URL
YOUTRACK_TOKEN
```

Configuration and credentials are stored separately. Permanent tokens are not written into the normal profile config.

```bash
youtrack auth status
youtrack auth logout
```

## Issue inspection

```bash
youtrack issue view TT-123
youtrack issue view TT-123 --json

youtrack issue field list TT-123
youtrack issue field get TT-123 Priority --plain
```

## Issue search and browser handoff

```bash
youtrack issue search 'project: APP'
youtrack issue search 'project: APP #Unresolved sort by: updated desc' --limit 20
youtrack issue search 'project: APP' --offset 100 --limit 50
youtrack issue search 'project: APP' --all
youtrack issue search 'project: APP #Unresolved' --json

youtrack issue open APP-123 --print-url --plain
youtrack issue search 'project: APP #Unresolved' open --print-url --plain
youtrack saved-search open 'Assigned to me' --print-url --plain
```

The browser-search grammar is exactly `issue search <query> open`. `issue search 'open'` remains an ordinary REST search for the literal query; `issue search 'open' open` opens that query in the browser. Browser-search mode does not accept `--limit`, `--offset`, or `--all`.

`--print-url` is the headless workflow: it prints the canonical context-safe URL and does not dispatch a browser. With no `--print-url`, `--json` and `--plain` select output formatting but still dispatch the platform browser opener. Direct `issue search <query> open` builds the URL without a token, credential read, or HTTP request. `issue open` and `saved-search open` authenticate only to resolve canonical issue or saved-query metadata; browser login and page authorization remain separate.

URLs preserve a configured context path such as `/youtrack`, encode query and issue values once, and never contain tokens. A successful OS dispatch is not evidence that the browser navigated or authenticated; an opener failure is not a mutation and must not be retried as one.

YouTrack performs filtering and sorting; the CLI preserves the quoted query and any explicit sort clause. Prefer bounded discovery with `--limit`; reserve `--all` for a genuinely complete collection.

## Saved searches

```bash
youtrack saved-search view 'Assigned to me'
youtrack saved-search view 51-33 --limit 20
youtrack saved-search view 'Release blockers' --all
```

`saved-search view` runs the visible server-side saved query. Result flags page its matching issues; the command is view-only. `saved-search open` instead opens the saved search's current query snapshot without fetching matching issues.

## Comments

```bash
youtrack issue comment list TT-123 --limit 20 --json
youtrack issue comment add TT-123 --text 'Short update'
youtrack issue comment add TT-123 --file ./comment.md
youtrack issue comment edit TT-123 4-17 --text 'Revised update'
youtrack issue comment edit TT-123 4-17 --file ./revised-comment.md
youtrack issue comment remove TT-123 4-17
```

Comment lists use bounded pagination by default (50); use `--all` only for a complete scan. Prefer `--file` for substantial multiline text because its bytes are preserved exactly. Edit replaces only the selected comment text; it is distinct from reversible `deleted=true` removal. Restoration, permanent deletion, attachments, visibility, reactions, and pinning remain unsupported.


## Issue time

```text
youtrack issue time types <issue> [--limit <n>] [--offset <n>] [--all]
youtrack issue time list <issue> [--limit <n>] [--offset <n>] [--all]
youtrack issue time view <issue> <work-item-id>
youtrack issue time add <issue> --duration <period> --date <YYYY-MM-DD>
  [--text <text> | --file <path>] [--type <type>] [--author <user>]
youtrack issue time edit <issue> <work-item-id>
  [--duration <period>] [--date <YYYY-MM-DD>]
  [--text <text> | --file <path>] [--type <type> | --clear-type] [--author <user>]
youtrack issue time remove <issue> <work-item-id> --yes
```

```bash
youtrack issue time types TT-123 --all --json
youtrack issue time list TT-123 --limit 20 --json
youtrack issue time view TT-123 115-7 --json
youtrack issue time add TT-123 --duration 1h30m --date 2026-09-29 --type Development --text 'Implement token refresh'
youtrack issue time edit TT-123 115-7 --duration 2h --text '' --clear-type --author alice
youtrack issue time remove TT-123 115-7 --yes
```

`types` and `list` accept `[--limit <positive-n>] [--offset <non-negative-n>] [--all]`; defaults are 50/0 and `--all` conflicts with an explicit limit. `types` discovers the selected issue project's types. `view` uses a work-item database ID. `--duration` is a required add value and a replacement edit value; use a positive integer `h`/`m` period such as `45m`, `2h`, or `1h30m`. `--date` is a required add value and a replacement edit value in `YYYY-MM-DD`. `--text` and `--file` are mutually exclusive and preserve exact bytes, including empty text. `--type` resolves a project-scoped type; `--clear-type` explicitly clears it and conflicts with `--type`. `--author` resolves an ID, login, or `@me`; `creator` is server-managed. Add/edit replace a work item value; they never increment a total custom field. `remove --yes` is a mandatory acknowledgement for permanent deletion.

## Issue relationships

```text
youtrack issue link types [--limit <n>] [--offset <n>] [--all]
youtrack issue link list <issue> --type <reference> [--direction outward|inward] [--limit <n>] [--offset <n>] [--all]
youtrack issue link add <issue> <target-issue> --type <reference> [--direction outward|inward]
youtrack issue link remove <issue> <target-issue> --type <reference> [--direction outward|inward]
```

```bash
youtrack issue link types --all --json
youtrack issue link add APP-CHILD APP-PARENT --type 'subtask of' --json
youtrack issue link list APP-CHILD --type 'subtask of' --all --json
youtrack issue link add APP-123 APP-125 --type 'relates to'
youtrack issue link remove APP-CHILD APP-PARENT --type 'subtask of'
```

Discover link types before selecting one. `--type` accepts a database ID, unique name, or unique configured relation label; it never assumes English aliases. A directed type selected by ID/name requires `--direction outward|inward`; a label supplies its direction and rejects a conflicting flag. Undirected types reject `--direction`. Type and linked-issue lists page with the normal defaults (offset 0, limit 50; `--all` fully scans). Add-existing and remove-missing are successful no-ops reported by JSON `present`/`changed`. Remove deletes only the selected edge, never an issue. Adding a subtask parent does not implicitly detach an existing parent or reparent an issue.

## Issue creation

```bash
youtrack issue create APP \
  --summary 'Login fails after token rotation' \
  --description-file ./description.md \
  --field Type=Bug \
  --field 'Fix versions=2026.2' \
  --field 'Fix versions=2026.3' \
  --field Assignee=@me \
  --tag backend \
  --json
```

`issue create` resolves the project, fields, and tags before one `POST /api/issues`. `--description` and `--description-file` are mutually exclusive; supplied description bytes are preserved exactly. Each `--field` splits on its first `=`, so commas are literal, and repeat a multi-value field to supply all values. `Board` and every state field are unavailable during creation. The returned issue uses the existing full issue output contract.

## Custom fields

```bash
youtrack issue field set TT-123 Priority Critical
youtrack issue field set TT-123 State "In Progress"
youtrack issue field set TT-123 Assignee @me
youtrack issue field clear TT-123 Assignee
```

Multi-value fields take multiple positional values:

```bash
youtrack issue field set TT-123 "Fix versions" 2026.2 2026.3
```

Supported MVP1 semantic kinds include string, integer, float, date, date-time, period, text, enum, state, user, version, build, owned, and group fields.

Periods support `m` and `h`, for example `30m` and `1h30m`.

## Combined edit

```bash
youtrack issue edit TT-123 \
  --summary "Improve authentication" \
  --description-file ./description.md \
  --field Priority=Critical \
  --field "State=In Progress" \
  --tag backend \
  --remove-tag legacy
```

For multi-value fields, repeat the field:

```bash
youtrack issue edit TT-123 \
  --field "Fix versions=2026.2" \
  --field "Fix versions=2026.3"
```

All supplied references are resolved before mutation. When the requested changes can be represented by one YouTrack issue update, the CLI sends one mutation request.

## Board membership

`Board` is a reserved synthetic multi-value field. It lists the issue's agile-board memberships; values are board database IDs or names, never sprint syntax. Setting it replaces the complete set, and YouTrack selects a board's current/default sprint:

```bash
youtrack issue field get TT-123 Board --json
youtrack issue field set TT-123 Board "Platform Board" "Operations Board"
youtrack issue field clear TT-123 Board
youtrack issue edit TT-123 --field "Board=Platform Board" --field "Board=Operations Board"
```

Board removals use YouTrack command parsing and can be rejected for query-driven sprint-disabled boards. The CLI validates the complete command first. A mixed edit validates all values first, then sends the ordinary issue update before the Board command; these two endpoints are not atomic, so an execution-time Board failure leaves a successful ordinary update intact. A real custom field named Board is addressable only by its project-field ID.

## Tags

```bash
youtrack issue tag list TT-123
youtrack issue tag add TT-123 backend
youtrack issue tag remove TT-123 backend
```

## Move an issue

```bash
youtrack issue move TT-123 PLATFORM
```

Project movement is deliberately separate from generic issue editing.

## Raw REST API

For endpoints not yet modeled by the CLI:

```bash
youtrack api /api/users/me

youtrack api /api/issues/TT-123 \
  --method POST \
  --data '{"summary":"Updated"}'
```

Only relative `/api` paths are accepted and the Authorization header cannot be overridden.

## Output

Normal output is human-oriented. `--json` returns the CLI's stable output model; it does not expose YouTrack REST DTOs. `--plain` is intended for shell pipelines.

```bash
youtrack issue view TT-123 --json
youtrack issue field get TT-123 Priority --plain
```

## Agent skill

The binary embeds an agent skill:

```bash
youtrack skill list
youtrack skill install
youtrack skill install --project
youtrack skill update
youtrack skill remove
```

The skill teaches coding agents to inspect before mutation, discover custom fields instead of guessing, and prefer structured commands over raw REST calls.

## Development

```bash
mise run fmt
mise run lint
mise run test
mise run test:race
mise run coverage
mise run ci
```

Tests use Testify assertions and handwritten fakes. REST tests use `httptest.Server`.

- [Release instructions](RELEASE.md)

## Design/specification

- [Overview](spec/mvp1/00-overview.md)
- [Architecture](spec/mvp1/01-architecture.md)
- [Custom fields](spec/mvp1/02-custom-fields.md)
- [CLI contract](spec/mvp1/03-cli-contract.md)
- [YouTrack REST adapter](spec/mvp1/04-youtrack-api.md)
- [Config/auth/output](spec/mvp1/05-config-auth-output.md)
- [Agent skill](spec/mvp1/06-agent-skill.md)
- [Testing/release](spec/mvp1/07-testing-release.md)
