# Library API and IR

Dobl can be used as a Go library.

```go
log, err := dobl.Parse(r)
if err != nil {
    // handle error
}

events := log.Events
steps := log.Steps()
```

## Event IR

`BuildLog.Events` is the primary intermediate representation. It preserves
input order and keeps the original input line in `Event.Raw`.

Event kinds:

- `step_start`
- `step_status`
- `step_output`
- `unknown`

Event statuses:

- `DONE`
- `CACHED`
- `ERROR`
- `CANCELED`
- `WARNING`
- `PROGRESS`

`PROGRESS` is assigned by the parser for BuildKit progress lines that are not a
terminal status.

Duration handling:

- `duration` preserves the original duration text, such as `0.4s` or `250ms`.
- `duration_nanos` is emitted when Go's `time.ParseDuration` can parse the
  duration.
- `0.0s` is represented as `duration_nanos: 0`, so zero duration remains
  distinguishable from a missing duration.

## Step Summary IR

`BuildLog.Steps()` groups events by BuildKit step ID in first-seen order.

Each `Step` includes:

- `id`
- `order`
- `name`
- `display_name`
- `category`
- `status`
- `duration`
- `duration_nanos`
- `stage`
- `index`
- `total`
- `instruction`
- `output_count`
- `progress_count`
- `unknown_count`
- `error_detail`
- `start_line`
- `end_line`
- optional source `events`

Dockerfile step metadata is extracted from names such as:

- `[1/2] FROM ...`
- `[build 1/3] RUN ...`
- `[stage-1 2/2] COPY ...`

Internal BuildKit steps such as `[internal] load metadata ...` and export
steps are not treated as Dockerfile step metadata.

## Visualization Metadata

Step summaries include a small set of additive fields intended for reports and
visualizations:

- `order` is the first-seen summary order, starting at 1.
- `display_name` is a UI-oriented label. Dockerfile step prefixes such as
  `[build 1/3]` are removed, while non-Dockerfile step names are preserved.
- `category` groups steps into broad reporting buckets:
  - `dockerfile`
  - `internal`
  - `export`
  - `cache`
  - `provenance`
  - `other`

`name` remains the parsed BuildKit step name and should be used when a rawer
label is needed.
