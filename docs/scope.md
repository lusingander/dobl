# Scope and Parser Behavior

Dobl currently targets non-streaming plain text logs from:

- `docker build --progress=plain`
- `docker buildx build --progress=plain`

It does not currently parse `--progress=rawjson`.

## Parser Behavior

The parser is intentionally conservative. BuildKit plain output is
human-readable text, not a stable machine protocol, so unknown lines are kept as
`unknown` events instead of being dropped.

Before classifying a line, the parser normalizes common terminal and CI
artifacts:

- ANSI control sequences are stripped.
- Carriage-return redraws are reduced to the last segment.
- Leading RFC3339 timestamps are accepted.
- Kubernetes/CRI-style stream prefixes such as `stdout F` and `stderr F` are
  accepted after a timestamp.
- Leading spaces and tabs before a BuildKit step line are ignored.

`Event.Raw` still preserves the original input line.

## Fixture Coverage

Fixtures under `testdata/` cover:

- successful builds
- cached builds
- command failures
- warnings
- cancellation
- metadata resolution failures
- interleaved/parallel BuildKit output
- CI/log-collector prefixes

## Known Limits

- Parsing is non-streaming and stores events in memory.
- Very long single input lines can exceed the scanner limit and return an
  error.
- `--progress=rawjson` is not implemented.
- Terminal, HTML, or other visualization output is not implemented.

## Development Notes

Use:

```sh
GOCACHE=/tmp/dobl-go-build go test ./...
```

The explicit `GOCACHE` avoids sandbox write failures from the default Go build
cache under the home directory in restricted environments.
