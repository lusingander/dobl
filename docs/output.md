# Output Formats

The default output format is JSON.

## Event JSON

`dobl parse --compact testdata/error_plain.log` emits line-oriented events:

```json
{"events":[{"line":1,"kind":"step_start","raw":"#1 [internal] load build definition from Dockerfile","step_id":"#1","detail":"[internal] load build definition from Dockerfile"},{"line":2,"kind":"step_status","raw":"#1 transferring dockerfile: 109B done","step_id":"#1","detail":"transferring dockerfile: 109B done","status":"PROGRESS"},{"line":3,"kind":"step_status","raw":"#1 DONE 0.0s","step_id":"#1","status":"DONE","duration":"0.0s","duration_nanos":0},{"line":6,"kind":"step_start","raw":"#3 [1/1] RUN echo before && exit 1","step_id":"#3","detail":"[1/1] RUN echo before && exit 1"},{"line":7,"kind":"step_output","raw":"#3 0.102 before","step_id":"#3","detail":"0.102 before"},{"line":9,"kind":"step_status","raw":"#3 ERROR: process \"...\" did not complete successfully: exit code: 2","step_id":"#3","detail":"process \"...\" did not complete successfully: exit code: 2","status":"ERROR"}]}
```

The example above is shortened for readability. Actual output includes every
input line, including `unknown` events.

Event fields:

- `line`: 1-based input line number.
- `kind`: `step_start`, `step_status`, `step_output`, or `unknown`.
- `raw`: original input line.
- `step_id`: BuildKit step ID such as `#3`, when present.
- `detail`: meaningful text after the step ID when not represented elsewhere.
- `status`: `DONE`, `CACHED`, `ERROR`, `CANCELED`, `WARNING`, or parser-created
  `PROGRESS`.
- `duration`: original duration text when present.
- `duration_nanos`: parsed duration in nanoseconds when Go can parse it.

## Summary JSON

`dobl summary --compact testdata/error_plain.log` emits derived step summaries:

```json
[{"id":"#1","order":1,"name":"[internal] load build definition from Dockerfile","display_name":"[internal] load build definition from Dockerfile","category":"internal","status":"DONE","duration":"0.0s","duration_nanos":0,"output_count":0,"progress_count":1,"warning_count":0,"unknown_count":0,"start_line":1,"end_line":3},{"id":"#2","order":2,"name":"[internal] load metadata for docker.io/library/alpine:3.20","display_name":"[internal] load metadata for docker.io/library/alpine:3.20","category":"internal","status":"DONE","duration":"0.4s","duration_nanos":400000000,"output_count":0,"progress_count":0,"warning_count":0,"unknown_count":0,"start_line":4,"end_line":5},{"id":"#3","order":3,"name":"[1/1] RUN echo before && exit 1","display_name":"RUN echo before && exit 1","category":"dockerfile","status":"ERROR","index":1,"total":1,"instruction":"RUN","output_count":2,"output_tail":["0.102 before","0.103 /bin/sh: exit: line 1: illegal number: 1"],"progress_count":0,"warning_count":0,"unknown_count":0,"error_detail":"process \"/bin/sh -c echo before && exit 1\" did not complete successfully: exit code: 2","start_line":6,"end_line":9}]
```

Summary fields:

- `id`: BuildKit step ID.
- `order`: first-seen summary order, starting at 1.
- `name`: first parsed step name for the ID.
- `display_name`: UI-oriented step label. Dockerfile step prefixes such as
  `[build 1/3]` are removed.
- `category`: broad reporting category. One of `dockerfile`, `internal`,
  `export`, `cache`, `provenance`, or `other`.
- `status`: latest parsed status for the step.
- `duration`: latest parsed duration text for the step.
- `duration_nanos`: parsed duration in nanoseconds when available.
- `stage`: Dockerfile stage name when parsed from the step name.
- `index`: Dockerfile step index when parsed.
- `total`: Dockerfile step total when parsed.
- `instruction`: Dockerfile instruction when parsed.
- `output_count`: number of `step_output` events for the step.
- `output_tail`: latest `step_output.detail` values for the step, capped to
  the last 5 lines. It is omitted when the step has no output events. Use
  `summary --events` when every source event is needed.
- `progress_count`: number of parser-created `PROGRESS` status events.
- `warning_count`: number of `WARNING` status events for the step.
- `unknown_count`: number of unknown events assigned to the step.
- `error_detail`: latest error detail for `ERROR` steps.
- `warning_detail`: latest warning detail for `WARNING` steps.
- `start_line`: first line for the step.
- `end_line`: last line for the step.
- `events`: source events, only included with `dobl summary --events`.

## Summary JSON Contract

Summary JSON is intended to be the stable input format for downstream reports
and visualizations. Consumers should treat the fields documented in this
section as the public contract. A machine-readable schema is available at
[`docs/summary.schema.json`](summary.schema.json).

Compatibility rules:

- Existing documented fields are additive API surface and should not be renamed
  or removed without a major compatibility decision.
- New fields may be added in future releases.
- Consumers should ignore unknown fields.
- Fields tagged with `omitempty` are absent when the parser has no value for
  them.
- Array order is meaningful. Steps are emitted in first-seen BuildKit step ID
  order, which is also stored in `order`.
- `events` is intentionally omitted by default to keep summary output compact.
  Use `dobl summary --events` when event-level replay or debugging is needed.

Field semantics for visualization:

- Use `id` as the stable BuildKit step identifier within one parsed log.
- Use `order` for timeline ordering.
- Use `name` when the full parsed BuildKit label is needed.
- Use `display_name` as the preferred compact UI label.
- Use `category` for grouping and coloring.
- Use `status` as the latest parsed status for the step.
- Use `error_detail` and `warning_detail` for highlighted diagnostics.
- Use `output_tail` for lightweight command output context. It contains at
  most the latest 5 output lines for the step.
- Use `start_line` and `end_line` to link a summary item back to the source
  log.

Status values:

- `DONE`
- `CACHED`
- `ERROR`
- `CANCELED`
- `WARNING`
- `PROGRESS`

Category values:

- `dockerfile`
- `internal`
- `export`
- `cache`
- `provenance`
- `other`

Duration values:

- `duration` preserves the original BuildKit duration text.
- `duration_nanos` is present only when the duration can be parsed by Go's
  `time.ParseDuration`.
- A parsed zero duration is emitted as `"duration_nanos": 0`.

## Summary Table

`dobl summary --format table testdata/error_plain.log` emits:

```text
ID  STATUS  DURATION  STEP  INSTRUCTION  NAME                                                        OUTPUTS  PROGRESS  ERROR
#1  DONE    0.0s                         [internal] load build definition from Dockerfile            0        1
#2  DONE    0.4s                         [internal] load metadata for docker.io/library/alpine:3.20  0        0
#3  ERROR             1/1   RUN          [1/1] RUN echo before && exit 1                             2        0         process "/bin/sh -c echo before && exit 1" did not complete successfully: exit code: 2
```

Columns:

- `ID`: BuildKit step ID.
- `STATUS`: latest parsed status.
- `DURATION`: latest parsed duration.
- `STEP`: Dockerfile stage/index summary, such as `build 1/3` or `1/1`.
- `INSTRUCTION`: Dockerfile instruction when parsed.
- `NAME`: full parsed step name.
- `OUTPUTS`: output event count.
- `PROGRESS`: progress event count.
- `ERROR`: error detail. Long values are truncated unless `--wide` is used.

## Category Rules

Summary `category` values are assigned from the parsed step name:

- `dockerfile`: Dockerfile step metadata such as `[build 1/3] RUN ...`.
- `internal`: names beginning with `[internal] `.
- `export`: names beginning with `exporting to `.
- `cache`: names beginning with `exporting cache to ` or
  `importing cache manifest from `.
- `provenance`: names beginning with `resolving provenance for `.
- `other`: anything else.
