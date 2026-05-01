# Dobl Summary Viewer

`index.html` is a static viewer for `dobl summary --format json` output. It has
no build step and no external runtime dependencies.

Generate summary JSON:

```sh
dobl summary --compact build.log > summary.json
```

Open `index.html` in a browser and load `summary.json`.

For local fixture data:

```sh
GOCACHE=/tmp/dobl-go-build go run ../../cmd/dobl summary --compact ../../testdata/visualization_contract_plain.log > sample-summary.json
```

The checked-in `sample-summary.json` is generated from
`testdata/visualization_contract_plain.log`.
