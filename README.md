# Dobl

Dobl parses Docker BuildKit `--progress=plain` build logs into structured JSON
and step summaries.

It is useful when CI logs need to be searched, filtered, summarized, or passed
to later reporting tools.

> [!NOTE]
> This project is primarily developed with OpenAI Codex. Most implementation, testing, and documentation work is delegated to Codex, with human review and direction.

## Install

```sh
go install github.com/lusingander/dobl/cmd/dobl@latest
```

## Quick Start

Parse a plain BuildKit log into line-oriented event JSON:

```sh
docker buildx build --progress=plain . 2>&1 | dobl parse
```

Summarize a saved log by BuildKit step:

```sh
dobl summary build.log
```

Show a human-readable table:

```sh
dobl summary --format table build.log
```

Show a richer static terminal summary:

```sh
dobl summary --format text build.log
```

Sort the summary for triage:

```sh
dobl summary --sort duration --format text build.log
```

Call out top triage targets without reordering the full step list:

```sh
dobl summary --top slow --format text build.log
```

Generate a self-contained HTML report:

```sh
dobl report build.log > report.html
```

Write a report directly and set its title:

```sh
dobl report --title "CI build" --output report.html build.log
```

Show only failed steps:

```sh
dobl summary --failed --format table build.log
```

Filter by Dockerfile metadata or step ID:

```sh
dobl summary --stage build --instruction RUN build.log
dobl summary --step '#3' --format table build.log
```

## Examples

`dobl summary --format table testdata/error_plain.log`:

```text
ID  STATUS  DURATION  STEP  INSTRUCTION  NAME                                                        OUTPUTS  PROGRESS  DIAGNOSTIC
#1  DONE    0.0s                         [internal] load build definition from Dockerfile            0        1
#2  DONE    0.4s                         [internal] load metadata for docker.io/library/alpine:3.20  0        0
#3  ERROR             1/1   RUN          [1/1] RUN echo before && exit 1                             2        0         process "/bin/sh -c echo before && exit 1" did not complete successfully: exit code: 2
```

`dobl summary --format text testdata/error_plain.log`:

```text
Dobl Summary
Source: testdata/error_plain.log
Steps: 3  Done: 2  Cached: 0  Warnings: 0  Errors: 1  Canceled: 0  Outputs: 2

Timeline:
#1 D internal 0.0s | #2 D internal 0.4s | #3 E RUN

Problems:
x  #3  ERROR  RUN  process "/bin/sh -c echo before && exit 1" did not complete successfully: exit code: 2
```

The default output format is JSON. Table output truncates long error details by
default; use `--wide` to keep full error text. Text output is a static
terminal-friendly view for CI logs and local triage, with optional top sections
for slow, warning-heavy, or noisy steps.

Summary JSON is the stable input contract for downstream reports and
visualizations. See [Output formats](docs/output.md) and the
[summary JSON schema](docs/summary.schema.json) for the documented fields.

A static summary viewer is available at [examples/viewer](examples/viewer).

## Development

Common development commands are available through
[Task](https://taskfile.dev/):

```sh
task build
task lint
task test
task browser:test
task ci
```

Install Node dependencies before running browser checks locally:

```sh
task npm:install
task browser:install
```

Regenerate the checked-in static viewer after editing
`internal/cli/viewer.html`:

```sh
task generate
```

GitHub Actions references are pinned to full commit SHAs with `pinact`:

```sh
task pinact:install
task pinact:update
task pinact:check
task pinact:verify
```

## Documentation

- [CLI reference](docs/cli.md)
- [Output formats](docs/output.md)
- [Library API and IR](docs/library.md)
- [Scope and parser behavior](docs/scope.md)

## Scope

Implemented:

- non-streaming `docker build` / `docker buildx build` plain progress logs
- event JSON output
- step summary JSON output
- step summary table output
- summary filters for status, failure, warning, stage, instruction, and step ID
- static summary JSON viewer
- self-contained HTML report output
- normalization for common ANSI, carriage-return, timestamp, and CI log-prefix
  artifacts

Not implemented:

- streaming parsing
- `--progress=rawjson`
- terminal visualization

## License

MIT
