# Configuration, Authentication, and Output

## Configuration

Use `github.com/BurntSushi/toml` for decoding and encoding.

Default config:

```text
$XDG_CONFIG_HOME/youtrack/config.toml
# fallback ~/.config/youtrack/config.toml
```

Example:

```toml
current = "company"

[profiles.company]
url = "https://company.youtrack.cloud"
denied_projects = ["SECRET", "Internal Finance"]
denied_tags = ["confidential", "personal-data"]
```

Tokens are not stored in the ordinary config.

### Profile access restrictions

`denied_projects` and `denied_tags` are optional string arrays. Projects match ID, short name, or full name; tags match ID or name. Match exact references case insensitively after trimming configured reference whitespace, without wildcard or substring semantics. Deny an issue when its project matches OR any tag matches. Omitted/empty lists impose no restrictions. Reject blank entries and malformed array values when loading/saving configuration.

`config set profile.<name>.denied_projects/denied_tags <json-array>` replaces a list; `[]` clears it. `config get` renders a compact JSON array; `config list` exposes lists in human/JSON output. Existing config keys and unrestricted behavior remain compatible.

The CLI wires the selected profile's policy into the application service, including when URL/token flags or environment overrides are supplied. Check issue access using lightweight project metadata and, when tag restrictions apply, a completely paginated issue-tag collection. Denials use the existing auth/permission exit code 3; metadata failures fail closed. Do not expose full issue details or perform mutations after a denied preflight. Check both link endpoints, source and destination projects for moves, and resolved create/edit/add-tag inputs before writing. Server workflows or concurrent changes are not transactional with these checks: post-write denials suppress issue output without rolling back successful writes.

Filter search, saved-search view, and linked-issue results before applying their offset/limit; continue scanning actual server pages, retain query text and sorting, and never return a partial result on metadata failure. Query pagination therefore counts allowed issues. Tag restrictions introduce additional metadata calls per candidate issue.

Disable raw API and browser query/saved-search handoff whenever either list is nonempty, including `--print-url`; those routes cannot enforce local filtering. Individual issue handoff checks policy first. Authentication and global link-type discovery remain available. Re-login preserves restriction lists. These are editable local guardrails, not server authorization or a restriction on independently using the browser or another profile.

## Credential store

MVP1 uses a separate TOML credential file with mode `0600`:

```text
~/.config/youtrack/credentials.toml
```

It is hidden behind `CredentialStore` so a future Keychain/Secret Service backend does not change CLI commands.

## Runtime precedence

```text
explicit CLI flags
> environment
> selected profile
> defaults
```

Environment:

```text
YOUTRACK_PROFILE
YOUTRACK_URL
YOUTRACK_TOKEN
```

An explicit server URL must not silently reuse credentials from a different stored profile. Provide an explicit token with a URL override.

## Login

`auth login` obtains URL/profile/token from flags/env/config or prompts, verifies them with `/api/users/me`, then writes profile + credential and selects that profile. Terminal token entry is hidden with `x/term`.

## Stable output model

Domain structs do not contain JSON tags. `internal/output/model.go` maps domain models into explicit compatibility-sensitive JSON DTOs.

The stable issue JSON includes readable ID, entity ID, summary/description, project, semantic fields, and tags. REST `$type` is not the primary JSON contract.
