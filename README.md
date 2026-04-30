# Dobl

Dobl parses plain Docker BuildKit build logs into JSON that can be inspected,
summarized, and eventually visualized.

The current target is `docker build` / `docker buildx build` output produced
with `--progress=plain`.

## Usage

Parse a build log into line-oriented events:

```sh
docker buildx build --progress=plain . 2>&1 | dobl parse
```

Read from a file:

```sh
dobl parse build.log
```

Summarize events by BuildKit step:

```sh
dobl summary build.log
```

Emit a human-readable summary table:

```sh
dobl summary --format table build.log
```

Include each step's source events in the summary:

```sh
dobl summary --events build.log
```

Emit compact JSON:

```sh
dobl parse --compact build.log
dobl summary --compact build.log
```

The default format is `json`. `dobl parse` currently supports `--format json`;
`dobl summary` supports `--format json` and `--format table`.

## Library

```go
log, err := dobl.Parse(r)
if err != nil {
    // handle error
}

events := log.Events
steps := log.Steps()
```

`Parse` keeps unknown lines as `unknown` events instead of dropping them. This
is intentional because BuildKit plain output is human-readable text and may
vary across Docker versions or CI environments.

The plain parser strips ANSI control sequences before classification, handles
carriage-return progress redraws, and accepts leading ISO-8601 UTC timestamps
commonly added by CI log collectors. `Event.Raw` still preserves the original
input line.

## Current IR

- `BuildLog.Events` preserves the original line order.
- `Event.Raw` keeps the original line.
- `Event.Kind` is one of `step_start`, `step_status`, `step_output`, or
  `unknown`.
- `Event.Status` uses typed constants for `DONE`, `CACHED`, `ERROR`,
  `CANCELED`, `WARNING`, and parser-generated `PROGRESS`.
- `Event.Duration` preserves the original duration text; `Event.DurationNanos`
  contains the parsed duration in nanoseconds when parsing succeeds.
- `Event.Detail` keeps the meaningful text after the BuildKit step ID when it
  is not already represented by status or duration. For example, it stores step
  names, progress text, error/warning messages, and command output.
- `BuildLog.Steps()` groups events by BuildKit step id in first-seen order.
  Each step includes output, progress, and unknown event counts, plus
  `error_detail` for the latest error status with detail text.
- Dockerfile step names such as `[build 1/3] RUN ...` are summarized into
  `stage`, `index`, `total`, and `instruction` fields when present.

## Scope

Implemented:

- non-streaming `--progress=plain` parsing
- event JSON output
- step summary JSON output
- step summary table output
- fixtures for success, cache, error, warning, cancellation, metadata failure,
  and interleaved BuildKit logs

Not implemented yet:

- streaming parsing
- `--progress=rawjson`
- terminal or HTML visualization
