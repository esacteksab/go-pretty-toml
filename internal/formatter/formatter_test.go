// SPDX-License-Identifier: MIT
package formatter

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestFormatTomlValue(t *testing.T) {
	testCases := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"string", "hello", `"hello"`},
		{"int", 123, "123"},
		{"float", 123.45, "123.45"},
		{"bool_true", true, "true"},
		{"bool_false", false, "false"},
		{"nil", nil, "''"},
		{"time", time.Date(2023, 1, 10, 15, 4, 5, 0, time.UTC), "2023-01-10T15:04:05Z"},
		{"simple_array", []interface{}{1, "a", true}, `[1, "a", true]`},
		{"empty_array", []interface{}{}, `[]`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatTomlValue(tc.input)
			if got != tc.want {
				t.Errorf("formatTomlValue(%#v) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// Helper type to simulate write errors
type errorWriter struct {
	err error
}

func (ew *errorWriter) Write(p []byte) (n int, err error) {
	return 0, ew.err // Always return the configured error
}

func TestFormat(t *testing.T) {
	// Define a specific error for testing write errors
	errSimulatedWriteFailed := errors.New("simulated write failure")

	testCases := []struct {
		name               string
		inputData          map[string]interface{}
		indentUnit         string
		outputWriter       io.Writer // Allow specifying writer for error testing
		wantOutput         string    // Expected output if no error
		wantErr            bool      // Should an error be returned?
		wantErrMsgContains string    // Substring expected in the error message
	}{
		// --- Successful Cases ---
		{
			name:         "simple_no_indent",
			inputData:    map[string]interface{}{"key": "value", "number": 100},
			indentUnit:   "",
			outputWriter: nil,
			wantOutput:   "key    = \"value\"\nnumber = 100\n",
			wantErr:      false,
		},
		{
			name: "table_with_indent",
			inputData: map[string]interface{}{
				"a":     1,
				"table": map[string]interface{}{"b": true, "c": "inside"},
			},
			indentUnit:   "  ",
			outputWriter: nil,
			wantOutput:   "a = 1\n\n[table]\n  b = true\n  c = \"inside\"\n",
			wantErr:      false,
		},
		{
			name: "array_table_with_indent",
			inputData: map[string]interface{}{
				"arr": []interface{}{
					map[string]interface{}{"x": 1},
					map[string]interface{}{"y": 2, "z": 3},
				},
			},
			indentUnit:   "  ",
			outputWriter: nil,
			wantOutput:   "[[arr]]\n  x = 1\n\n[[arr]]\n  y = 2\n  z = 3\n",
			wantErr:      false,
		},
		{
			name: "nested_tables_indent",
			inputData: map[string]interface{}{
				"server": map[string]interface{}{
					"ip":    "1.1.1.1",
					"ports": map[string]interface{}{"http": 80},
				},
			},
			indentUnit:   "\t",
			outputWriter: nil,
			wantOutput:   "[server]\n\tip = \"1.1.1.1\"\n\n\t[ports]\n\t\thttp = 80\n",
			wantErr:      false,
		},
		{
			name:         "empty_map",
			inputData:    map[string]interface{}{},
			indentUnit:   " ",
			outputWriter: nil,
			wantOutput:   "",
			wantErr:      false,
		},

		// --- Error Cases ---
		{
			name: "error_bad_array_item",
			inputData: map[string]interface{}{
				"key_before": "value", // Add a key before to ensure simple key processing happens first
				"bad_arr": []interface{}{
					map[string]interface{}{"a": 1}, // Good item
					"not a map",                    // Bad item - should cause error
					map[string]interface{}{"b": 2}, // Item after bad one (shouldn't be processed)
				},
				"key_after": "value2", // Add a key after
			},
			indentUnit: "", outputWriter: nil,
			wantOutput:         "", // Output might be partial ("key_before = ..."), safer not to check exact output on error
			wantErr:            true,
			wantErrMsgContains: "key 'bad_arr': arrays cannot mix tables and non-tables",
		},
		{
			name:               "error_write_failed",
			inputData:          map[string]interface{}{"key": "value"}, // Valid data needed
			indentUnit:         "",
			outputWriter:       &errorWriter{err: errSimulatedWriteFailed},
			wantOutput:         "",
			wantErr:            true,
			wantErrMsgContains: errSimulatedWriteFailed.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var writer io.Writer
			var buf *bytes.Buffer // Keep track if we used a buffer

			if tc.outputWriter != nil {
				writer = tc.outputWriter
			} else {
				buf = &bytes.Buffer{} // Use a buffer for successful cases
				writer = buf
			}

			err := Format(tc.inputData, tc.indentUnit, writer)

			// Check if error occurred when expected/unexpected
			if tc.wantErr {
				if err == nil {
					// If error was expected but nil, print buffer content for debugging
					debugOutput := "<writer was not a buffer>"
					if buf != nil {
						debugOutput = buf.String()
					}
					t.Fatalf(
						"Format() expected an error, but got nil. Output buffer content:\n%s",
						debugOutput,
					)
				}
				if tc.wantErrMsgContains != "" {
					if !strings.Contains(err.Error(), tc.wantErrMsgContains) {
						t.Errorf(
							"Format() error = %q, want error containing %q",
							err,
							tc.wantErrMsgContains,
						)
					}
				}
			} else { // Should succeed
				if err != nil {
					t.Fatalf("Format() returned unexpected error: %v", err)
				}
				if buf != nil { // Only check output if we used a buffer
					gotOutput := buf.String()
					if gotOutput != tc.wantOutput {
						t.Errorf("Format() output mismatch:\ngot:\n%s\nwant:\n%s", gotOutput, tc.wantOutput)
					}
				}
			}
		})
	}
}
