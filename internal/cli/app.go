package cli

import (
	"fmt"
	"io"

	"github.com/alecthomas/kong"
)

type application struct {
	Parse   parseCmd   `cmd:"" help:"Parse a plain build log into event JSON."`
	Summary summaryCmd `cmd:"" help:"Summarize a plain build log by BuildKit step."`
	Report  reportCmd  `cmd:"" help:"Generate a self-contained HTML summary report."`
}

type runContext struct {
	stdin  io.Reader
	stdout io.Writer
}

type kongExit int

func Run(args []string, stdin io.Reader, stdout io.Writer) (err error) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		exitCode, ok := recovered.(kongExit)
		if !ok {
			panic(recovered)
		}
		if exitCode != 0 {
			err = fmt.Errorf("exit %d", exitCode)
		}
	}()

	var app application
	parser, err := kong.New(
		&app,
		kong.Name("dobl"),
		kong.Description("Parse and summarize plain Docker BuildKit build logs."),
		kong.Writers(stdout, io.Discard),
		kong.Exit(func(code int) {
			panic(kongExit(code))
		}),
		kong.Bind(&runContext{stdin: stdin, stdout: stdout}),
	)
	if err != nil {
		return err
	}

	ctx, err := parser.Parse(args[1:])
	if err != nil {
		return err
	}

	return ctx.Run()
}
