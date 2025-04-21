// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

// --- Testscript setup ---

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"toml-fmt": main,
	})
}

// TestScripts runs all *.txtar files in the testdata directory (no change)
func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata",
	})
}

// Helper to temporarily set os.Args for testing kingpin/flag parsing
func setTestArgs(args []string) func() {
	originalArgs := os.Args
	os.Args = append([]string{"toml-fmt"}, args...) // Prepend program name
	return func() {
		os.Args = originalArgs // Restore original args after test
	}
}

func TestParseArgsAndGetInput(t *testing.T) {
	// Create a dummy file for testing file input
	tmpDir := t.TempDir()
	dummyFilePath := filepath.Join(tmpDir, "dummy.toml")
	err := os.WriteFile(dummyFilePath, []byte("test=1"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create dummy test file: %v", err)
	}

	testCases := []struct {
		name           string
		args           []string
		wantIndentUnit string
		wantWrite      bool
		wantFilename   string // Expected cleaned filename
		wantSourceName string
		wantInputType  string // "stdin" or "file"
		wantErr        bool
	}{
		{"stdin_no_flags", []string{}, "", false, "", "stdin", "stdin", false},
		{"stdin_indent", []string{"-i"}, "  ", false, "", "stdin", "stdin", false},
		{
			"file_no_flags",
			[]string{dummyFilePath},
			"",
			false,
			dummyFilePath,
			"file '" + dummyFilePath + "'",
			"file",
			false,
		},
		{
			"file_indent",
			[]string{"-i", dummyFilePath},
			"  ",
			false,
			dummyFilePath,
			"file '" + dummyFilePath + "'",
			"file",
			false,
		},
		{
			"file_write",
			[]string{"-w", dummyFilePath},
			"",
			true,
			dummyFilePath,
			"file '" + dummyFilePath + "'",
			"file",
			false,
		},
		{
			"file_write_indent",
			[]string{"-i", "-w", dummyFilePath},
			"  ",
			true,
			dummyFilePath,
			"file '" + dummyFilePath + "'",
			"file",
			false,
		},
		{
			"file_write_indent_alt_order",
			[]string{dummyFilePath, "-i", "-w"},
			"  ",
			true,
			dummyFilePath,
			"file '" + dummyFilePath + "'",
			"file",
			false,
		}, // kingpin allows this
		{"error_write_stdin", []string{"-w"}, "", true, "", "", "", true},
		{
			"error_file_not_exist",
			[]string{"nonexistent.toml"},
			"",
			false,
			"nonexistent.toml",
			"file 'nonexistent.toml'",
			"file",
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restoreArgs := setTestArgs(tc.args)
			defer restoreArgs()

			reader, indent, write, fname, sname, err := parseArgsAndGetInput()

			if (err != nil) != tc.wantErr {
				t.Fatalf("parseArgsAndGetInput() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil {
				return // Don't check other fields if error was expected
			}

			// Clean up the reader if it's a file
			if closer, ok := reader.(io.Closer); ok && reader != os.Stdin {
				defer closer.Close()
			}

			if indent != tc.wantIndentUnit {
				t.Errorf("indentUnit got = %q, want %q", indent, tc.wantIndentUnit)
			}
			if write != tc.wantWrite {
				t.Errorf("writeToFile got = %v, want %v", write, tc.wantWrite)
			}
			if fname != tc.wantFilename {
				t.Errorf("filename got = %q, want %q", fname, tc.wantFilename)
			}
			if sname != tc.wantSourceName {
				t.Errorf("sourceName got = %q, want %q", sname, tc.wantSourceName)
			}

			inputType := "stdin"
			if reader != os.Stdin {
				inputType = "file"
			}
			if inputType != tc.wantInputType {
				t.Errorf("Input type appears to be %q, want %q", inputType, tc.wantInputType)
			}
		})
	}
}

func TestWriteOutput(t *testing.T) {
	content := "formatted = true\n"
	contentBytes := []byte(content)

	t.Run("write_to_stdout", func(t *testing.T) {
		contentBuf := bytes.NewBuffer(contentBytes) // Fresh buffer for each subtest
		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := writeOutput(false, "", contentBuf)
		w.Close()             // Close writer to signal EOF to reader
		os.Stdout = oldStdout // Restore stdout

		if err != nil {
			t.Fatalf("writeOutput to stdout returned error: %v", err)
		}

		capturedBytes, _ := io.ReadAll(r)
		if string(capturedBytes) != content {
			t.Errorf("stdout got = %q, want %q", string(capturedBytes), content)
		}
	})

	t.Run("write_to_file", func(t *testing.T) {
		contentBuf := bytes.NewBuffer(contentBytes) // Fresh buffer
		tmpDir := t.TempDir()
		targetFilePath := filepath.Join(tmpDir, "output.toml")

		err := writeOutput(true, targetFilePath, contentBuf)
		if err != nil {
			t.Fatalf("writeOutput to file returned error: %v", err)
		}

		// Check file content
		fileBytes, err := os.ReadFile(targetFilePath)
		if err != nil {
			t.Fatalf("Failed to read back target file: %v", err)
		}
		if string(fileBytes) != content {
			t.Errorf("File content got = %q, want %q", string(fileBytes), content)
		}
	})

	t.Run("write_to_file_empty_buffer", func(t *testing.T) {
		contentBuf := &bytes.Buffer{} // Fresh empty buffer
		tmpDir := t.TempDir()
		targetFilePath := filepath.Join(tmpDir, "empty_output.toml")
		err := os.WriteFile(targetFilePath, []byte("initial content"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create initial file: %v", err)
		}

		err = writeOutput(true, targetFilePath, contentBuf)
		if err != nil {
			t.Fatalf("writeOutput(empty) to file returned error: %v", err)
		}

		fileBytes, err := os.ReadFile(targetFilePath)
		if err != nil {
			t.Fatalf("Failed to read back empty target file: %v", err)
		}
		if len(fileBytes) != 0 {
			t.Errorf("File content should be empty, got %d bytes", len(fileBytes))
		}
	})
}
