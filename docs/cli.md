# CLI Reference

Dobl has four commands:

- `dobl parse [file]`
- `dobl summary [file]`
- `dobl report [file]`
- `dobl tui [file]`

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
dobl summary --format text build.log
dobl summary --format table --wide build.log
dobl summary --format text --wide build.log
dobl summary --failed --format table build.log
dobl summary --failed --format text build.log
dobl summary --warnings --format table build.log
dobl summary --status ERROR build.log
dobl summary --sort duration --format text build.log
dobl summary --top slow --format text build.log
dobl summary --details all --format text build.log
dobl summary --stage build --instruction RUN build.log
dobl summary --step '#3' build.log
```

Flags:

- `--format json|table|text`
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
- `--sort KEY`
  - Sort steps. The default is `order`.
  - Supported keys: `order`, `duration`, `status`, `outputs`, `warnings`.
  - `duration`, `outputs`, and `warnings` sort descending. `status` sorts
    problem statuses first.
- `--top KEY`
  - Include a top section in text output. Only supported with `--format text`.
  - Supported keys: `slow`, `warnings`, `outputs`.
- `--details MODE`
  - Set the text detail section mode. Only supported with `--format text`.
  - Supported modes: `problems`, `all`, `none`. The default is `problems`.
- `--wide`
  - Do not truncate table or text diagnostics. Only supported with
    `--format table` or `--format text`.
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
dobl report --output report.html build.log
dobl report --title "CI build" --output report.html build.log
docker buildx build --progress=plain . 2>&1 | dobl report > report.html
```

The report embeds the same step summary JSON used by `dobl summary --format
json` into the static viewer UI. It can be opened directly in a browser and
does not require a server.

Flags:

- `-o`, `--output FILE`
  - Write the report to a file instead of stdout.
- `--title TITLE`
  - Set the report title shown in the HTML viewer.
- `-h`, `--help`
  - Show command help.

## `dobl tui`

Inspect a completed plain BuildKit build log in an interactive terminal UI.

```sh
dobl tui [file]
```

Examples:

```sh
dobl tui build.log
dobl tui --filter problems build.log
dobl tui --search "missing dependency" build.log
dobl summary --compact build.log > summary.json
dobl tui --summary summary.json
docker buildx build --progress=plain . 2>&1 | dobl tui
```

The TUI uses the same parsed step summary model as `dobl summary --format json`.
It starts with completed, non-streaming logs. Use `--summary` to inspect an
existing summary JSON array instead of parsing a build log. When input is read
from stdin, stdin is consumed as the build log or summary JSON and keyboard
input is read from the terminal. Terminal output is required; `TERM=dumb` and
redirected TUI output are rejected.

Keyboard controls:

- `j`, `k`, arrow keys
  - Move the selected step when the steps pane is focused.
  - Scroll details when the details pane is focused.
- `tab`
  - Switch focus between the steps and details panes.
- `g`, `G`
  - Jump to the first or last visible step.
- `pageup`, `pagedown`, `ctrl+u`, `ctrl+d`
  - Scroll the selected step detail panel.
- `n`, `N`
  - Move to the next or previous visible problem step.
- `f`
  - Cycle filters: all, problems, warnings, failed.
- `p`
  - Switch directly to the problems filter.
- `r`
  - Reset the filter and search query.
- `/`
  - Search steps by ID, status, category, instruction, name, diagnostics, or
    output tail.
- `esc`
  - Clear the current search.
- `q`, `ctrl+c`
  - Quit.

Flags:

- `--summary FILE`
  - Read summary JSON from this file instead of parsing a plain build log.
  - Use `--summary -` to read summary JSON from stdin.
- `--filter all|problems|warnings|failed`
  - Set the initial filter. The default is `all`.
- `--search QUERY`
  - Set the initial search query.
- `-h`, `--help`
  - Show command help.

## Validation

Invalid flag combinations are rejected before reading input:

- `--failed` and `--status` cannot be used together.
- `--warnings` and `--status` cannot be used together.
- `--failed` and `--warnings` cannot be used together.
- `--events` is only supported with `--format json`.
- `--compact` is only supported with `--format json`.
- `--top` is only supported with `--format text`.
- `--details` is only supported with `--format text`.
- `--wide` is only supported with `--format table` or `--format text`.
