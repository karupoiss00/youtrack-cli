# Commands

```bash
youtrack issue search <query> [--limit <n>] [--offset <n>] [--all]
youtrack issue search <query> open [--print-url] [--json|--plain]
youtrack issue open <issue> [--print-url] [--json|--plain]
youtrack saved-search view <saved-search> [--limit <n>] [--offset <n>] [--all]
youtrack saved-search open <saved-search> [--print-url] [--json|--plain]
youtrack issue comment list <issue> [--limit <n>] [--offset <n>] [--all]
youtrack issue comment add <issue> (--text <text> | --file <path>)
youtrack issue comment edit <issue> <comment-entity-id> (--text <text> | --file <path>)
youtrack issue comment remove <issue> <comment-entity-id>
youtrack issue time types <issue> [--limit <n>] [--offset <n>] [--all]
youtrack issue time list <issue> [--limit <n>] [--offset <n>] [--all]
youtrack issue time view <issue> <work-item-id>
youtrack issue time add <issue> --duration <period> --date <YYYY-MM-DD> [--text <text> | --file <path>] [--type <type>] [--author <user>]
youtrack issue time edit <issue> <work-item-id> [--duration <period>] [--date <YYYY-MM-DD>] [--text <text> | --file <path>] [--type <type> | --clear-type] [--author <user>]
youtrack issue time remove <issue> <work-item-id> --yes
youtrack issue link types [--limit <n>] [--offset <n>] [--all]
youtrack issue link list <issue> --type <reference> [--direction outward|inward] [--limit <n>] [--offset <n>] [--all]
youtrack issue link add <issue> <target-issue> --type <reference> [--direction outward|inward]
youtrack issue link remove <issue> <target-issue> --type <reference> [--direction outward|inward]




youtrack issue view <issue> [--json|--plain]

youtrack issue create <project> \
  --summary TEXT \
  [--description TEXT | --description-file FILE] \
  [--field NAME=VALUE ...] \
  [--tag TAG ...]

youtrack issue edit <issue> \
  [--summary TEXT] \
  [--description TEXT | --description-file FILE] \
  [--field NAME=VALUE ...] \
  [--tag TAG ...] \
  [--remove-tag TAG ...]

youtrack issue move <issue> <project>

youtrack issue field list <issue>
youtrack issue field get <issue> <field>
youtrack issue field set <issue> <field> <value>...
youtrack issue field clear <issue> <field>

youtrack issue tag list <issue>
youtrack issue tag add <issue> <tag>
youtrack issue tag remove <issue> <tag>

youtrack api <endpoint> [--method METHOD] [--data JSON | --data-file FILE]

youtrack config list [--json]
youtrack config get profile.<name>.denied_projects
youtrack config get profile.<name>.denied_tags
youtrack config set profile.<name>.denied_projects '["SECRET","Internal Finance"]'
youtrack config set profile.<name>.denied_tags '["private","personal-data"]'
```

Profile `denied_projects` and `denied_tags` are TOML string arrays; `config set` accepts JSON arrays and `[]` clears a list. Projects match ID/short name/full name and tags match ID/name, exactly and case insensitively. Configured reference whitespace is trimmed. A matching project or any matching tag denies the task with exit code 3. Missing/empty lists preserve unrestricted behavior. Read-only `config list` displays active lists, including in JSON.

Checks cover issue reads/opening, fields, tags, comments, time entries, moves, creation, and both ends of link changes. Denied destinations and newly assigned denied tags are rejected before writes. Search, saved-search view, and linked-issue lists omit denied tasks; offsets/limits count allowed tasks, and server sorting stays intact. Tag checks paginate every issue tag and may require extra requests; metadata failures stop output. Raw `api` and both browser search commands are disabled under restrictions, including URL-only mode. Issue browser opening remains available after an access check. URL/token overrides retain the selected profile's rules, and re-login preserves them. These are local guardrails; YouTrack permissions, server workflows, concurrent changes, and direct browser access remain independent. A failure after a server-side change does not roll it back; inspect uncertain mutations before retrying.

`issue search` accepts one non-blank opaque query for its REST mode. Quote it so YouTrack receives its filters and explicit sort clause unchanged. It starts at offset `0` and returns `50` results by default. `--limit` must be positive, `--offset` must be non-negative, and `--all` cannot be combined with an explicitly supplied `--limit`. Use `--` when the query begins with a hyphen:

```bash
youtrack issue search -- '-State: Done project: APP'
```

The exact lowercase suffix selects browser mode: `issue search <query> open`. Therefore `issue search 'open'` remains a literal REST query, while `issue search 'open' open` opens it. Browser mode does not accept pagination flags. `--print-url` prints the context-safe destination without dispatching a browser; for unattended use, prefer `--print-url --json`. `--json` or `--plain` without `--print-url` only changes output formatting and still dispatches. Direct browser search does not read credentials or contact YouTrack.

`issue open` resolves a canonical readable issue ID, and `saved-search open` resolves the saved search's current query snapshot, then dispatches or prints its URL. Both require ordinary API authentication for that metadata; page authentication/navigation is still performed by the browser. URLs never contain tokens. OS dispatch acceptance is not page-render success.

`saved-search view` accepts one database ID or quoted visible name. It resolves the visible server-side saved query, then pages matching issues with the same defaults and validation as REST `issue search`: offset `0`, limit `50`, a positive explicit `--limit`, non-negative `--offset`, and no explicit `--limit` with `--all`. Use `--json` for the saved-search metadata and issue summaries, or `--plain` for matching issue lines only. The command is view-only; structured saved-search create, update, delete, and sharing operations are unavailable.

`issue comment list` uses the same pagination rules as `issue search`: default limit 50, non-negative offset, positive explicit limit, and no explicit `--limit` with `--all`. It includes visible soft-removed comments. `issue comment add` and `issue comment edit` require exactly one source selected with `--text` or `--file`; both preserve raw supplied bytes, and blank/whitespace-only text is rejected. Edit sends a text-only POST to the selected comment and does not restore a removed comment. Prefer `--file` for substantial multiline content. `issue comment remove` performs reversible soft removal only. Restoration, permanent deletion, attachments, visibility, reactions, and pinning are unsupported. Use `--json` for stable comment objects, `--plain` for IDs (and no bytes on successful removal), or human output for readable blocks.

`issue time types` discovers project-scoped work-item types and `issue time list` uses the ordinary pagination defaults: offset 0, limit 50, positive explicit `--limit`, non-negative `--offset`, and no explicit limit with `--all`. `add` requires an explicit `YYYY-MM-DD` date and a positive integer `h`/`m` duration such as `45m`, `2h`, or `1h30m`. `--text` and `--file` are mutually exclusive and preserve exact bytes, including empty text. `--author` resolves a user ID/login or `@me`; the creator is server-managed. `edit` replaces only supplied values, preserves omitted values, and uses `--clear-type` to send an explicit type clear. Work-item IDs are database IDs. `remove --yes` permanently deletes after an item read; it emits no plain bytes, while JSON reports `{\"entityId\":...,\"removed\":true}`. Do not blindly retry a failed or uncertain mutation.

`issue link types` discovers server-configured relationship types. `list` uses the ordinary pagination rules: offset 0, limit 50, positive explicit limit, non-negative offset, and no explicit limit with `--all`. `--type` is required for list/add/remove and accepts a type ID, unique type name, or unique configured outward/inward label. A directed type selected by ID/name requires exactly `--direction outward|inward`; a label derives that direction and rejects a conflict. An undirected type rejects a direction. Add-existing and remove-missing are successful no-ops; JSON reports final `present` and `changed`. Add/remove first resolve both issue database IDs and fully scan the selected edge collection, then perform at most one write. Removal deletes only the edge; adding a subtask parent does not implicitly detach a prior parent or write a reciprocal edge.

`issue create` accepts exactly one project and a non-blank `--summary`. `--description` and `--description-file` are mutually exclusive and preserve supplied bytes. Each `--field` is split only on its first `=`, so commas and later equals signs remain literal; repeat a multi-value field to provide its complete value set. The CLI resolves the project, all fields, and all tags before issuing one non-retried issue POST, then renders the returned full issue. `Board`, state fields, attachments, links, comments, and notification inputs are unavailable during creation.

`Board` is available through the field commands and `--field Board=...`. Board set is exact-set replacement; values are board IDs or names, not sprint syntax.

Global connection flags: `--profile`, `--url`, `--token`, `--timeout`. Global output flags: `--json`, `--plain`.
