package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/lusingander/dobl"
)

func main() {
	if err := run(os.Args, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) < 2 || args[1] != "parse" {
		return errors.New("usage: dobl parse [file]")
	}
	if len(args) > 3 {
		return errors.New("usage: dobl parse [file]")
	}

	var input io.Reader = stdin
	var file *os.File
	if len(args) == 3 && args[2] != "-" {
		var err error
		file, err = os.Open(args[2])
		if err != nil {
			return err
		}
		defer file.Close()
		input = file
	}

	log, err := dobl.Parse(input)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(log)
}
