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

Generate a self-contained HTML report:

```sh
dobl report build.log > report.html
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

The default output format is JSON. Table output truncates long error details by
default; use `--wide` to keep full error text.

Summary JSON is the stable input contract for downstream reports and
visualizations. See [Output formats](docs/output.md) and the
[summary JSON schema](docs/summary.schema.json) for the documented fields.

A static summary viewer is available at [examples/viewer](examples/viewer).

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
