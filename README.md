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

Show only failed steps:

```sh
dobl summary --failed build.log
dobl summary --failed --format table build.log
```

Filter steps by status:

```sh
dobl summary --status ERROR build.log
dobl summary --status WARNING --format table build.log
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

## Output Examples

`dobl parse --compact build.log` emits line-oriented events:

```json
{"events":[{"line":1,"kind":"step_start","raw":"#1 [internal] load build definition from Dockerfile","step_id":"#1","detail":"[internal] load build definition from Dockerfile"},{"line":3,"kind":"step_status","raw":"#1 DONE 0.0s","step_id":"#1","status":"DONE","duration":"0.0s","duration_nanos":0},{"line":7,"kind":"step_output","raw":"#3 0.102 before","step_id":"#3","detail":"0.102 before"},{"line":9,"kind":"step_status","raw":"#3 ERROR: process \"...\" did not complete successfully: exit code: 2","step_id":"#3","detail":"process \"...\" did not complete successfully: exit code: 2","status":"ERROR"}]}
```

`dobl summary --compact build.log` emits derived step summaries:

```json
[{"id":"#3","name":"[1/1] RUN echo before && exit 1","status":"ERROR","index":1,"total":1,"instruction":"RUN","output_count":2,"progress_count":0,"unknown_count":0,"error_detail":"process \"...\" did not complete successfully: exit code: 2","start_line":6,"end_line":9}]
```

`dobl summary --format table build.log` emits a readable table:

```text
ID  STATUS  DURATION  STEP  INSTRUCTION  OUTPUTS  PROGRESS  ERROR
#1  DONE    0.0s                         0        1
#2  DONE    0.4s                         0        0
#3  ERROR             1/1   RUN          2        0         process "/bin/sh -c echo before && exit 1" did not complete successfully: exit code: 2
```

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

## License

MIT
