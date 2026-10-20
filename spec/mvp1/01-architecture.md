# Architecture

## Repository layout

```text
youtrack-cli/
├── cmd/youtrack/main.go
├── internal/
│   ├── cli/
│   ├── app/
│   ├── domain/
│   ├── youtrack/
│   │   └── dto/
│   ├── config/
│   ├── credentials/
│   ├── output/
│   ├── skill/
│   └── version/
├── skills/youtrack-cli/
├── spec/mvp1/
├── integration/
├── testdata/
├── mise.toml
├── .golangci.yml
├── .goreleaser.yaml
├── AGENTS.md
└── README.md
```

## Dependency direction

```text
cli -> app -> domain
youtrack -> app ports + domain
output -> domain
config, credentials, skill -> independent utilities
```

Forbidden dependencies:

- domain -> CLI/config/YouTrack
- app -> REST DTOs
- CLI -> REST DTOs
- output -> REST DTOs

Interfaces are declared by their consumer in `internal/app/ports.go`.

## Main application ports

`IssueStore` provides read/update/move/tag operations. `ProjectFieldStore` provides project field definitions, bundle values, and allowed field users. `ProjectStore`, `TagStore`, `UserStore`, and `GroupStore` provide reference resolution. `BoardStore` resolves semantic Board memberships and validates/applies Board command changes.

Application services receive those interfaces and never instantiate HTTP/config clients themselves.

`IssueAccessStore` supplies lightweight project metadata and the complete paginated tag set when requested. `AccessPolicy` belongs to the application layer; the CLI derives it from the selected config profile and injects it into `Service`. The YouTrack adapter owns the metadata request and tag pagination. Every issue operation checks policy before content reads/writes, with resolved destination/tag checks before mutations. Restricted issue collections share application filtering before offset/limit; REST DTOs remain inside the adapter and normal JSON output contracts are unchanged. Unrestricted commands avoid access-metadata requests.

## Mutation model

Application code builds an `IssuePatch` containing summary, description, resolved custom-field assignments, and an optional complete desired tag set. Board membership is a semantic multi-valued synthetic field and remains outside `IssuePatch`; its adapter owns sprint/agile and command wire shapes.

Combined edit flow:

```text
GET issue + issue sprints
 -> load/cache project metadata
 -> resolve every custom field, Board value, and tag
 -> validate Board command assist and all changes
 -> optional POST /api/issues/{issue}
 -> optional POST /api/commands for Board
 -> GET issue + issue sprints
```

Successful Board edits use the issue POST before the command POST. They are validation-safe but not transactionally atomic: a command execution failure does not roll back a successful issue POST.

A validation error before the POST must leave the issue untouched.

## Command-scoped caches

The field resolver caches within a command invocation:

- project field definitions
- bundle options
- allowed field users
- current authenticated user

No persistent cache is part of MVP1.

## Concurrency

Mutation operations are sequential. Do not parallelize writes. Metadata lookup optimization is secondary to predictable behavior.
