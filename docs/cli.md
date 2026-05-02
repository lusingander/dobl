# CLI Reference

Dobl has three commands:

- `dobl parse [file]`
- `dobl summary [file]`
- `dobl report [file]`

When `file` is omitted or set to `-`, input is read from stdin.

## Install

```sh
go install github.com/lusingander/dobl/cmd/dobl@latest
```

## `dobl parse`

Parse a plain BuildKit build log into event JSON.

```sh
dobl parse [file] [flags]
```

Examples:

```sh
dobl parse build.log
docker buildx build --progress=plain . 2>&1 | dobl parse --compact
```

Flags:

- `--format json`
  - Output format. `json` is currently the only supported parse format.
- `--compact`
  - Emit compact JSON instead of indented JSON.
- `-h`, `--help`
  - Show command help.

## `dobl summary`

Summarize a plain BuildKit build log by BuildKit step.

```sh
dobl summary [file] [flags]
```

Examples:

```sh
dobl summary build.log
dobl summary --format table build.log
dobl summary --format table --wide build.log
dobl summary --failed --format table build.log
dobl summary --warnings --format table build.log
dobl summary --status ERROR build.log
dobl summary --stage build --instruction RUN build.log
dobl summary --step '#3' build.log
```

Flags:

- `--format json|table`
  - Output format. The default is `json`.
- `--compact`
  - Emit compact JSON. Only supported with `--format json`.
- `--events`
  - Include each step's source events. Only supported with `--format json`.
- `--failed`
  - Include only failed steps. This includes `ERROR` and `CANCELED`, but not
    `WARNING`.
- `--warnings`
  - Include only warning steps. This includes steps with `WARNING` status or
    parsed warning events.
- `--status STATUS`
  - Include only steps with the given status.
  - Supported statuses: `DONE`, `CACHED`, `ERROR`, `CANCELED`, `WARNING`,
    `PROGRESS`.
- `--stage STAGE`
  - Include only Dockerfile steps from this parsed stage.
- `--instruction INSTRUCTION`
  - Include only Dockerfile steps with this instruction. Matching is
    case-insensitive.
- `--step ID`
  - Include only a specific BuildKit step ID. Both `#3` and `3` are accepted.
  - Malformed IDs such as `abc` are rejected.
- `--wide`
  - Do not truncate table error details. Only supported with `--format table`.
- `-h`, `--help`
  - Show command help.

## `dobl report`

Generate a self-contained HTML summary report.

```sh
dobl report [file]
```

Examples:

```sh
dobl report build.log > report.html
docker buildx build --progress=plain . 2>&1 | dobl report > report.html
```

The report embeds the same step summary JSON used by `dobl summary --format
json` into the static viewer UI. It can be opened directly in a browser and
does not require a server.

## Validation

Invalid flag combinations are rejected before reading input:

- `--failed` and `--status` cannot be used together.
- `--warnings` and `--status` cannot be used together.
- `--failed` and `--warnings` cannot be used together.
- `--events` is only supported with `--format json`.
- `--compact` is only supported with `--format json`.
- `--wide` is only supported with `--format table`.
