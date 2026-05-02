package main

import (
	"fmt"
	"os"

	"github.com/lusingander/dobl/internal/cli"
)

func main() {
	if err := cli.Run(os.Args, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
