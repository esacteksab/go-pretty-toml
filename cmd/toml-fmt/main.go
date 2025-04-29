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
	"github.com/esacteksab/go-pretty-toml/internal/version"
)

// writeOutput writes the formatted TOML content either to stdout or back to the original file.
// When writing to a file, it uses a safe approach with a temporary file and atomic rename.
//
// Parameters:
//   - writeToFile: Whether to write to the source file (true) or stdout (false)
//   - inputFilename: The source file path (must be non-empty if writeToFile is true)
//   - outputBuf: Buffer containing the formatted TOML content
//
// Returns:
//   - error: Any error encountered during the write operation, or nil on success
func writeOutput(writeToFile bool, inputFilename string, outputBuf *bytes.Buffer) error {
	if !writeToFile {
		// Write to stdout
		_, err := outputBuf.WriteTo(os.Stdout) // Write the buffer content to standard output
		if err != nil {
			return fmt.Errorf("writing to stdout: %w", err) // Wrap the error with context
		}
	} else {
		// Sanity check: filename should be non-empty when writing to file
		if inputFilename == "" {
			return errors.New("internal error: writeToFile is true but inputFilename is empty") // Return an error if the filename is empty when writing to file
		}

		// Create a temporary file in the same directory as the input file
		tempFile, err := os.CreateTemp(filepath.Dir(inputFilename), filepath.Base(inputFilename)+".tmp") // Create a temporary file in the same directory with a ".tmp" extension
		if err != nil {
			return fmt.Errorf("creating temporary file: %w", err) // Wrap the error with context
		}
		tempFilename := tempFile.Name() // Get the name of the temporary file

		// Track if rename succeeded to clean up temp file if needed
		renameSucceeded := false // Initialize the renameSucceeded flag
		defer func() {
			if !renameSucceeded {
				_ = os.Remove(tempFilename) // Remove the temporary file if the rename operation failed
			}
		}()

		// Write formatted content to temp file
		_, err = outputBuf.WriteTo(tempFile) // Write the formatted TOML content to the temporary file
		if err != nil {
			if closeErr := tempFile.Close(); closeErr != nil { // Try to close the temp file
				fmt.Fprintf(os.Stderr, "Warning: error closing temp file after write error: %v\n", closeErr) // Print a warning to stderr if closing fails
			}
			return fmt.Errorf("writing to temporary file '%s': %w", tempFilename, err) // Wrap the error with context
		}

		// Close temp file before rename
		err = tempFile.Close() // Close the temporary file before renaming it
		if err != nil {
			return fmt.Errorf("closing temporary file '%s': %w", tempFilename, err) // Wrap the error with context
		}

		// Atomically replace the original file with the temp file
		err = os.Rename(tempFilename, inputFilename) // Atomically rename the temporary file to the original filename, replacing the original
		if err != nil {
			return fmt.Errorf("renaming temporary file '%s' to '%s': %w", tempFilename, inputFilename, err) // Wrap the error with context
		}
		renameSucceeded = true // Set renameSucceeded to true if the rename was successful
	}
	return nil // Return nil if the write operation was successful
}

// getInput determines the input source (stdin or file) based on arguments.
// It opens the file if specified and returns an io.ReadCloser along with filename info.
//
// Parameters:
//   - filenameArg: The filename argument from command line (empty for stdin)
//   - writeToFile: Whether output should be written back to the source file
//
// Returns:
//   - inputReader: Reader for the input source (file or stdin)
//   - filename: Cleaned filename (empty for stdin)
//   - sourceName: Description of the source for error messages
//   - err: Any error encountered during setup, or nil on success
func getInput(
	filenameArg string,
	writeToFile bool,
) (inputReader io.ReadCloser, filename, sourceName string, err error) {
	if filenameArg == "" {
		// Reading from stdin
		if writeToFile {
			err = errors.New(
				"cannot use -w flag when reading from stdin",
			) // Return an error if the -w flag is used with stdin
			return
		}
		sourceName = "stdin"   // Set the source name to stdin
		inputReader = os.Stdin // os.Stdin is an *os.File, which is an io.ReadCloser. Assign standard input to the input reader.
	} else {
		// Reading from file
		filename = filepath.Clean(filenameArg)          // Clean the filename argument to remove any relative pathing
		sourceName = fmt.Sprintf("file '%s'", filename) // Set the source name to the filename
		var file *os.File
		file, err = os.Open(filename) //nolint:gosec // Open the file with the given filename
		if err != nil {
			err = fmt.Errorf("opening %s: %w", sourceName, err) // Wrap the error with context
			return
		}
		defer file.Close() //nolint:errcheck
		inputReader = file // Assign the opened file to the input reader
	}
	return // Return the determined reader, names, and nil error
}

// runFormattingLogic contains the core program logic after flag parsing.
// It handles input acquisition, TOML parsing, formatting, and output.
//
// Parameters:
//   - indentEnable: Whether to enable indentation in the formatted output
//   - writeToFile: Whether to write results back to source file (vs stdout)
//   - filenameArg: Input filename from command line (empty for stdin)
//
// Returns:
//   - error: Any error encountered during processing, or nil on success
func runFormattingLogic(indentEnable, writeToFile bool, filenameArg string) error {
	// Set indentation based on flag
	indentUnit := "" // Initialize the indent unit to an empty string
	if indentEnable {
		indentUnit = "  " // Set the indent unit to two spaces if indentation is enabled
	}

	// Get input source (stdin or file)
	inputReader, inputFilename, inputSourceName, err := getInput(
		filenameArg,
		writeToFile,
	) // Get the input reader, filename, and source name based on the command-line arguments
	if err != nil {
		return err // Return error from getInput (e.g., -w with stdin, file open error)
	}

	// Ensure the input reader is closed eventually (important for files)
	if closer, ok := inputReader.(io.Closer); ok &&
		inputReader != os.Stdin { // Check if the input reader implements the io.Closer interface and is not stdin
		defer func() { _ = closer.Close() }() // Schedule the input reader to be closed when the function returns
	}

	// Read All Input
	inputBytes, err := io.ReadAll(inputReader) // Read all the input from the input reader
	if err != nil {
		return fmt.Errorf(
			"reading from %s: %w",
			inputSourceName,
			err,
		) // Wrap the error with context
	}

	// Close input file *now* if writing back (to release file handle before potential write)
	if writeToFile &&
		inputReader != os.Stdin { // Check if the output is being written to a file and the input reader is not stdin
		if closer, ok := inputReader.(io.Closer); ok { // Check if the input reader implements the io.Closer interface
			// Ignore error on close here, as we've already read the content
			_ = closer.Close() // Close the input reader to release the file handle
		}
	}

	// Parse TOML
	var data map[string]interface{}         // Declare a variable to hold the parsed TOML data
	err = toml.Unmarshal(inputBytes, &data) // Parse the TOML data from the input bytes
	if err != nil {
		// Provide detailed parsing error if possible
		if docErr, ok := err.(*toml.DecodeError); ok { // Check if the error is a TOML decode error
			line, col := docErr.Position() // Get the line and column number of the error
			return fmt.Errorf("parsing TOML from %s at line %d, column %d: %w",
				inputSourceName, line, col, docErr) // Wrap the error with detailed context
		}
		return fmt.Errorf(
			"parsing TOML from %s: %w",
			inputSourceName,
			err,
		) // Wrap the error with context
	}

	// Handle empty input case gracefully
	if data == nil {
		emptyBuf := &bytes.Buffer{} // create an empty buffer
		// Pass inputFilename obtained from getInput
		err = writeOutput(
			writeToFile,
			inputFilename,
			emptyBuf,
		) // write the empty buffer to the output
		if err != nil {
			return fmt.Errorf("writing empty output: %w", err) // Wrap the error with context
		}
		return nil // Successful empty processing
	}

	// Format TOML Data
	var outputBuf bytes.Buffer // Declare a buffer to hold the formatted TOML data
	err = formatter.Format(
		data,
		indentUnit,
		&outputBuf,
	) // Format the TOML data using the formatter package
	if err != nil {
		return fmt.Errorf("formatting TOML data: %w", err) // Wrap the error with context
	}

	// Write Output
	err = writeOutput(
		writeToFile,
		inputFilename,
		&outputBuf,
	) // Write the formatted TOML data to the output
	if err != nil {
		return fmt.Errorf("writing output: %w", err) // Wrap the error with context
	}

	return nil // Success
}

// main is the entry point for the toml-fmt tool.
// It parses command-line arguments and orchestrates the formatting process.
func main() {
	// Define command-line application with description
	app := kingpin.New(
		"toml-fmt",
		"Formats TOML files with alignment and optional indentation.",
	) // Create a new Kingpin application
	app.HelpFlag.Short(
		'h',
	) // Set the short flag for the help flag
	app.Version(
		version.GetVersionInfo(),
	) // Set the version information for the application
	app.VersionFlag.Short(
		'v',
	) // Set the short flag for the version flag

	// Define flags and arguments
	writeToFile := app.Flag("write", "Write result back to the source file instead of stdout.").
		// Define the -w/--write flag
		Short('w').
		// Set the short flag
		Bool()
		// Set the type to boolean
	indentEnable := app.Flag("indent", "Indent output using two spaces.").
		Short('i').
		Bool()
		// Define the -i/--indent flag
	filenameArg := app.Arg("filename", "Input TOML file (optional, reads from stdin if omitted)").
		// Define the filename argument
		String()
		// Set the type to string

	// Parse arguments - kingpin handles errors/help/version automatically and exits
	kingpin.MustParse(app.Parse(os.Args[1:])) // Parse the command-line arguments

	// Run the core formatting logic with parsed arguments
	err := runFormattingLogic(
		*indentEnable,
		*writeToFile,
		*filenameArg,
	) // Run the core formatting logic with the parsed arguments
	// Handle any errors
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err) // Print the error message to stderr
		os.Exit(1)                                 // Exit with a non-zero exit code
	}

	// Exit cleanly if successful
	os.Exit(0) // Exit with a zero exit code
}
