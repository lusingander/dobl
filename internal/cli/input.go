package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/lusingander/dobl"
)

func parseInput(fileName string, stdin io.Reader) (*dobl.BuildLog, error) {
	if fileName == "-" {
		fileName = ""
	}

	var input io.Reader = stdin
	var file *os.File
	if fileName != "" {
		var err error
		file, err = os.Open(fileName)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", fileName, err)
		}
		defer file.Close()
		input = file
	}

	log, err := dobl.Parse(input)
	if err != nil {
		source := "stdin"
		if fileName != "" {
			source = fileName
		}
		return nil, fmt.Errorf("parse %s: %w", source, err)
	}
	return log, nil
}
