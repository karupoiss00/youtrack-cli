---
name: youtrack-cli
version: 0.3.0
description: Use when working with YouTrack issues, visible saved searches, comments, spent time, and issue relationships from a coding agent; drives the `youtrack` CLI for inspection, fields including Board membership, tags, project moves, comment management, issue time management, relationship management, and raw API access.
---

# YouTrack CLI (`youtrack`)

## Quick start

```bash
youtrack auth status
youtrack issue search 'project: APP #Unresolved' --limit 20 --json
youtrack issue open TT-123 --print-url --json
youtrack issue search 'project: APP #Unresolved' open --print-url --json
youtrack saved-search open 'Assigned to me' --print-url --json
youtrack issue create APP --summary 'Clear reproduction steps' --description-file ./description.md --json
youtrack saved-search view 'Assigned to me' --limit 20 --json
youtrack issue comment list TT-123 --limit 20 --json
youtrack issue field list TT-123 --json
youtrack issue time types TT-123 --all --json
youtrack issue time list TT-123 --limit 20 --json
youtrack issue link types --all --json
youtrack issue link list TT-123 --type 'relates to' --limit 20 --json
```

Do not guess command flags, custom-field names, enum/state/version values, or users. Use `youtrack <command> --help`, `issue field list`, and `issue field get` to inspect the current issue before mutating it.

## Rules

- Inspect an issue before changing it.
- Prefer structured commands over `youtrack api`.
- Honor profile `denied_projects` / `denied_tags`: a matching project or any matching tag denies the issue (exit 3). Do not change profiles or clear restrictions to bypass a denial. Raw API and browser search are disabled while either list is nonempty; use `issue search` or `saved-search view`. Search and link-list pagination counts allowed results. Configure lists only when requested, using JSON arrays with `config set profile.<name>.denied_projects/denied_tags`.
- Use `--json` when consuming results programmatically.
- Use `issue move` for project changes; do not try to edit the project as a normal field.
- Use `issue tag add/remove` for isolated tag changes.
- Use `issue edit` when several issue changes should be validated and applied together.
- Use `@me` only for user fields and `@none` to clear nullable fields.
- Re-read the issue after a meaningful mutation when verification matters.
- For `issue create`, choose the project deliberately, use a clear summary, use `--description-file` for substantial multiline content, resolve field and tag values rather than guessing, and inspect the returned issue when workflow defaults matter.
- Do not supply `Board` or any state field to `issue create`; creation does not accept attachments, links, comments, or notification inputs.
- Treat `Board` as a reserved multi-value field: inspect it first, use board IDs/names only, and remember that set replaces the full membership set.
- Do not invent sprint syntax. YouTrack selects the board's current/default sprint.
- A mixed normal-field and Board edit validates first but is not transactionally atomic if Board command execution later fails.
- Quote the single `issue search` query, use YouTrack's server-side search rather than local filtering, and preserve explicit server sorting.
- `issue search 'open'` is a REST search; add the exact lowercase suffix (`issue search 'open' open`) to hand it to the browser. Browser mode omits `--limit`, `--offset`, and `--all`.
- For unattended browser handoff, prefer `--print-url --json`; `--json` alone still dispatches the OS opener.
- Use `issue open` for a canonical issue route and `saved-search open` for a saved search's current query snapshot. Never put a token in a URL.
- Direct browser search constructs its URL without credentials or HTTP. Issue and saved-search browser routes authenticate for metadata only; OS dispatch success is distinct from browser login, navigation, and authorization.
- Use `saved-search view` with an existing visible saved search instead of recreating its server-side filter locally.
- Saved searches are view-only: do not use structured create, update, delete, or sharing operations.
- List comments before choosing a comment ID; use `--file` for substantial replacement text.
- Do not put permanent tokens in command logs.
- `issue comment edit` replaces text only; `issue comment remove` is reversible soft removal, not restoration or permanent deletion.
- Discover project work-item types with `issue time types` before adding or changing time; use explicit `YYYY-MM-DD` dates and positive `h`/`m` durations.
- Work-item IDs are database IDs. `issue time edit` replaces only supplied values; `--clear-type` explicitly clears a type, and `issue time remove <issue> <id> --yes` permanently deletes an item.
- Treat a failed or uncertain time mutation as uncertain; inspect before deciding whether to retry.
- Discover relationship types with `issue link types` before add/remove. Use the discovered type ID/name plus direction, or a unique configured label; do not assume English aliases.
- A link direction is relative to the first issue. Adding `APP-CHILD APP-PARENT --type 'subtask of'` sends one child-to-parent edge; it never adds a reciprocal edge or implicitly reparents.
- Treat a failed or uncertain relationship mutation as uncertain; inspect the selected relation before deciding whether to retry.

## Core commands

| Area | Commands |
| --- | --- |
| Auth | `auth login`, `auth logout`, `auth status` |
| Issue | `issue search`, `issue view`, `issue open`, `issue create`, `issue edit`, `issue move` |
| Comments | `issue comment list/add/edit/remove` |
| Issue time | `issue time types/list/view/add/edit/remove` |
| Issue links | `issue link types/list/add/remove` |
| Fields | `issue field list/get/set/clear` |
| Saved searches | `saved-search view`, `saved-search open` |
| API | `api <endpoint>` |
| Config | `config list/get/set` |

See `references/commands.md`, `references/fields.md`, and `references/workflows.md` for details.
