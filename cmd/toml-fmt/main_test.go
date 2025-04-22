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
