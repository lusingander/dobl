# Dobl

Dobl parses Docker BuildKit `--progress=plain` build logs into structured JSON,
step summaries, and self-contained reports.

It is intended for CI and local build logs that need to be searched, filtered,
summarized, or turned into a shareable report.

> [!NOTE]
> This project is primarily developed with OpenAI Codex. Most implementation,
> testing, and documentation work is delegated to Codex, with human review and
> direction.

## Install

```sh
go install github.com/lusingander/dobl/cmd/dobl@latest
```

## Quick Start

Capture a plain BuildKit log:

```sh
docker buildx build --progress=plain . 2>&1 | tee build.log
```

Summarize it in the terminal:

```sh
dobl summary --format text build.log
```

Generate a self-contained HTML report:

```sh
dobl report build.log > report.html
```

Produce JSON for other tools:

```sh
dobl summary --compact build.log > summary.json
```

For all commands, flags, filters, and output modes, see the
[CLI reference](docs/cli.md).

## What It Provides

- Event JSON from BuildKit plain progress logs.
- Step summary JSON with stable fields for downstream tools.
- Human-readable table and static terminal summaries.
- Filters for failed steps, warnings, statuses, Dockerfile metadata, and step
  IDs.
- A self-contained HTML report generated from a build log.
- A static summary JSON viewer in [examples/viewer](examples/viewer).

## Documentation

- [CLI reference](docs/cli.md)
- [Output formats](docs/output.md)
- [Library API and IR](docs/library.md)
- [Scope and parser behavior](docs/scope.md)
- [Summary JSON schema](docs/summary.schema.json)

## Development

Common checks are available through [Task](https://taskfile.dev/):

```sh
task ci
```

Useful focused commands:

```sh
task test
task lint
task browser:test
task generate
```

Install Node and browser dependencies before running browser checks locally:

```sh
task npm:install
task browser:install
```

## Scope

Implemented:

- non-streaming `docker build` / `docker buildx build` plain progress logs
- event JSON output
- step summary JSON, table, and text output
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
