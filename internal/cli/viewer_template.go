package cli

import _ "embed"

//go:generate cp viewer.html ../../examples/viewer/index.html
//go:embed viewer.html
var viewerHTML string
