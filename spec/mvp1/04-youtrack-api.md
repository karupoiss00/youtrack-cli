# YouTrack REST Adapter

## Client

`Client` owns normalized service URL, permanent token, `http.Client`, and user agent.

Every normal request sends:

```text
Authorization: Bearer <token>
Accept: application/json
User-Agent: youtrack-cli/<version>
Content-Type: application/json   # JSON mutations
```

Default timeout is 30 seconds and all requests honor `context.Context`.

## Base URL

Users configure the YouTrack service root, for example:

```text
https://company.youtrack.cloud
https://internal.example.com/youtrack
```

The adapter preserves any context path and appends `/api/...`.

## Read retries

GET only: retry 502/503/504 twice with short backoff. Do not automatically retry POST/DELETE in MVP1.

## Issue resources

```text
GET  /api/issues/{issue}
GET  /api/issues/{issue}/sprints
POST /api/issues/{issue}
POST /api/issues/{issue}/project
GET/POST /api/issues/{issue}/tags
DELETE /api/issues/{issue}/tags/{tagID}
POST /api/commands/assist
POST /api/commands
```

Combined `issue edit` serializes summary/description/customFields/final tags into one issue POST when present. Board membership uses sprint/agile reads, `/api/commands/assist` validation, and a command POST.

Access preflight requests only `id,project(id,name,shortName)` from the issue. With tag restrictions it additionally paginates `GET /api/issues/{issue}/tags?fields=id,name&$skip=...&$top=...` to exhaustion, advancing by the actual page size even when the server returns fewer than requested. The [official issue-tag resource](https://www.jetbrains.com/help/youtrack/devportal/resource-api-issues-issueID-tags.html) returns tags visible to the authenticated user; local policy cannot inspect hidden tags. Missing project identity and metadata/tag errors fail closed.

## Metadata resources

```text
GET /api/admin/projects/{project}
GET /api/admin/projects?query=...
GET /api/admin/projects/{project}/customFields
GET /api/admin/projects/{project}/customFields/{field}/bundle/values
GET /api/admin/customFieldSettings/bundles/user/{bundle}/aggregatedUsers
GET /api/agiles
GET /api/tags?query=...
GET /api/groups?query=...
GET /api/users/me
```

## Fields projections

All field projections are centralized in `internal/youtrack/fields_query.go`. Commands must not build YouTrack `fields=` expressions.

## REST DTOs

Read custom fields retain raw `value` JSON plus `$type`, and state-machine fields also request `possibleEvents(id,presentation)`, then map explicitly into domain values/semantic transitions. Board reads request `id,agile(id,name)` from issue sprints; Board metadata requests `id,name,projects(id)` from agiles. Do not build a large polymorphic DTO hierarchy.

Write DTOs are separate. The adapter maps `FieldDefinition.Kind + Cardinality` into the correct issue custom-field `$type`, then maps typed domain values into the required REST value representation. Board commands remain adapter-private and use `commands(error,description,delete)` during assist validation.

## Errors

`APIError` records status, method, path, YouTrack error code/description, request ID, and response body. Normal error strings are concise. Tokens and Authorization headers are never included.

HTTP 401/403 map to authentication errors, 404 to not-found, and other non-2xx responses to runtime/API errors.
