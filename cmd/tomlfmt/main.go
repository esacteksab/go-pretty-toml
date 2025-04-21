// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	kingpin "github.com/alecthomas/kingpin/v2"
	toml "github.com/pelletier/go-toml/v2"

	"github.com/esacteksab/go-pretty-toml/internal/formatter"
)

func parseArgsAndGetInput() (inputReader io.ReadCloser, indentUnit string, writeToFile bool, filename, sourceName string, err error) {
	app := kingpin.New("toml-fmt", "Formats TOML files with alignment and optional indentation.")
	app.HelpFlag.Short('h')

	kpWriteToFile := app.Flag("write", "Write result back to the source file instead of stdout.").
		Short('w').
		Bool()
	kpIndentEnable := app.Flag("indent", "Indent output using two spaces.").Short('i').Bool()
	kpFilenameArg := app.Arg("filename", "Input TOML file (optional, reads from stdin if omitted)").
		String()

	_, parseErr := app.Parse(os.Args[1:])
	if parseErr != nil {
		err = parseErr
		return
	}

	writeToFile = *kpWriteToFile
	filename = *kpFilenameArg
	indentUnit = ""
	if *kpIndentEnable {
		indentUnit = "  "
	}

	if filename == "" {
		if writeToFile {
			err = errors.New("cannot use -w flag when reading from stdin")
			return
		}
		sourceName = "stdin"
		inputReader = os.Stdin
	} else {
		filename = filepath.Clean(filename)
		sourceName = fmt.Sprintf("file '%s'", filename)
		var file *os.File
		file, err = os.Open(filename)
		if err != nil {
			err = fmt.Errorf("opening %s: %w", sourceName, err)
			return
		}
		inputReader = file
	}
	return
}

func writeOutput(writeToFile bool, inputFilename string, outputBuf *bytes.Buffer) error {
	if !writeToFile {
		_, err := outputBuf.WriteTo(os.Stdout)
		if err != nil {
			return fmt.Errorf("writing to stdout: %w", err)
		}
	} else {
		if inputFilename == "" {
			return errors.New("internal error: writeToFile is true but inputFilename is empty")
		}
		tempFile, err := os.CreateTemp(filepath.Dir(inputFilename), filepath.Base(inputFilename)+".tmp")
		if err != nil {
			return fmt.Errorf("creating temporary file: %w", err)
		}
		tempFilename := tempFile.Name()
		renameSucceeded := false
		defer func() {
			if !renameSucceeded {
				_ = os.Remove(tempFilename)
			}
		}()
		_, err = outputBuf.WriteTo(tempFile)
		if err != nil {
			if closeErr := tempFile.Close(); closeErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: error closing temp file after write error: %v\n", closeErr)
			}
			return fmt.Errorf("writing to temporary file '%s': %w", tempFilename, err)
		}
		err = tempFile.Close()
		if err != nil {
			return fmt.Errorf("closing temporary file '%s': %w", tempFilename, err)
		}
		err = os.Rename(tempFilename, inputFilename)
		if err != nil {
			return fmt.Errorf("renaming temporary file '%s' to '%s': %w", tempFilename, inputFilename, err)
		}
		renameSucceeded = true
	}
	return nil
}

func main() {
	// Parse Args and Get Input Source
	inputReader, indentUnit, writeToFile, inputFilename, inputSourceName, err := parseArgsAndGetInput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if closer, ok := inputReader.(io.Closer); ok && inputReader != os.Stdin {
		defer func() { _ = closer.Close() }()
	}

	// Read All Input
	inputBytes, err := io.ReadAll(inputReader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading from %s: %v\n", inputSourceName, err)
		os.Exit(1)
	}

	// Close input file *now* if writing back
	if writeToFile && inputReader != os.Stdin {
		if closer, ok := inputReader.(io.Closer); ok {
			_ = closer.Close()
		}
	}

	// Parse TOML
	var data map[string]interface{}
	err = toml.Unmarshal(inputBytes, &data)
	if err != nil {
		if docErr, ok := err.(*toml.DecodeError); ok {
			line, col := docErr.Position()
			fmt.Fprintf(
				os.Stderr,
				"Error parsing TOML from %s at line %d, column %d: %v\n",
				inputSourceName,
				line,
				col,
				docErr,
			)
		} else {
			fmt.Fprintf(os.Stderr, "Error parsing TOML from %s: %v\n", inputSourceName, err)
		}
		os.Exit(1)
	}

	// Handle empty input case gracefully
	if data == nil {
		if writeToFile {
			err = writeOutput(writeToFile, inputFilename, &bytes.Buffer{})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing empty output: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Println()
		}
		os.Exit(0)
	}

	// Format TOML Data
	var outputBuf bytes.Buffer
	// Call the exported function from the formatter package
	err = formatter.Format(data, indentUnit, &outputBuf) // Pass buffer as io.Writer
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting TOML data: %v\n", err)
		os.Exit(1)
	}

	// Write Output
	err = writeOutput(writeToFile, inputFilename, &outputBuf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}
}
