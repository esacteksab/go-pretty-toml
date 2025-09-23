// SPDX-License-Identifier: MIT

package formatter

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Format takes a map representing parsed TOML data and writes it to the provided
// output writer with proper formatting including alignment of values and optional
// indentation. Keys are sorted alphabetically and grouped by type.
//
// Parameters:
//   - data: Map representing parsed TOML data structure (map[string]interface{})
//   - indentUnit: String to use for each level of indentation (e.g. "" or "  ")
//   - output: Writer where formatted TOML will be written (io.Writer)
//
// Returns:
//   - error: If any formatting operation fails
func Format(data map[string]any, indentUnit string, output io.Writer) error {
	var internalBuf bytes.Buffer // Use a buffer to accumulate the formatted output
	// Start with an empty path for the root map. The path represents the nested structure of the TOML file.
	err := formatMap(data, []string{}, "", indentUnit, &internalBuf)
	if err != nil {
		return err
	}
	// Write the content of the buffer to the output writer
	_, err = internalBuf.WriteTo(output)
	return err
}

// formatTomlValue converts a Go value to its TOML string representation.
// Handles strings, integers, floats, booleans, times, nil values, and arrays.
//
// Parameters:
//   - v: The Go value to be converted to a TOML string
//
// Returns:
//   - string: TOML string representation of the value
func formatTomlValue(v any) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val) // Quote strings
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val) // Format integers
	case float32, float64:
		return fmt.Sprintf("%g", val) // Format floats using compact representation ("g" format is shortest representation)
	case bool:
		return strconv.FormatBool(val) // Convert boolean to "true" or "false"
	case time.Time:
		return val.Format(time.RFC3339Nano) // Format time in RFC3339 format (most precise)
	case nil:
		return "''" // Represent nil as empty quoted string
	case []any:
		// Handle arrays by formatting each element and joining with commas
		var elements []string
		for _, item := range val {
			elements = append(elements, formatTomlValue(item)) // Recursively format each element
		}
		return "[" + strings.Join(elements, ", ") + "]" // Join the elements with commas and enclose in square brackets
	default:
		return fmt.Sprintf("<<UNKNOWN TYPE %T>>", v) // Handle unknown types - returns a debug string
	}
}

// formatSimpleKeys formats and writes simple key-value pairs with proper alignment.
// Simple keys are those with non-table, non-array-table values.
//
// Parameters:
//   - dataMap: Map containing the key-value pairs
//   - simpleKeys: Slice of keys to process
//   - maxKeyLen: Maximum key length for alignment
//   - currentIndent: Current indentation string
//   - output: Buffer where formatted output is written
func formatSimpleKeys(
	dataMap map[string]any,
	simpleKeys []string,
	maxKeyLen int,
	currentIndent string, // Indent for the line itself
	output *bytes.Buffer,
) {
	for _, k := range simpleKeys {
		v := dataMap[k] // Get the value associated with the key
		padding := strings.Repeat(
			" ",
			maxKeyLen-len(k),
		) // Calculate padding for alignment
		formattedValue := formatTomlValue(
			v,
		) // Format the value into a TOML string
		fmt.Fprintf(
			output,
			"%s%s%s = %s\n",
			currentIndent,
			k,
			padding,
			formattedValue,
		) // Write the formatted key-value pair to the output buffer
	}
}

// formatArrayTables formats and writes array tables with proper headers and content.
// Array tables are represented as [[section.name]] in TOML.
//
// Parameters:
//   - arrayTableKeys: Map of keys to array tables
//   - currentPath: Current path to this section
//   - currentIndent: Current indentation string
//   - indentUnit: Unit of indentation for nested content
//   - output: Buffer where formatted output is written
//
// Returns:
//   - error: If any formatting operation fails
func formatArrayTables(
	arrayTableKeys map[string][]any,
	currentPath []string, // Path to the parent map
	currentIndent string,
	indentUnit string,
	output *bytes.Buffer,
) error {
	// Sort keys for consistent output
	sortedArrayTableKeys := make(
		[]string,
		0,
		len(arrayTableKeys),
	) // Create a slice to hold the sorted keys
	for k := range arrayTableKeys {
		sortedArrayTableKeys = append(sortedArrayTableKeys, k) // Add each key to the slice
	}
	sort.Strings(sortedArrayTableKeys) // Sort the keys alphabetically

	for _, k := range sortedArrayTableKeys {
		arrData := arrayTableKeys[k] // Retrieve the array of data for the key
		// Construct the full path for the array table key
		fullPath := append(append([]string{}, currentPath...), k) // Create copy before appending
		fullPathString := strings.Join(
			fullPath,
			".",
		) // Convert the path to a dot-separated string

		for i, item := range arrData {
			subMap, ok := item.(map[string]any) // Type assert each item as a map
			if !ok {
				return fmt.Errorf(
					"internal error: item in array table '%s' not a map",
					fullPathString,
				)
			}
			// Add newline separator between sections
			if output.Len() > 0 {
				trimmedLen := len(
					bytes.TrimRight(output.Bytes(), "\n\t "),
				) // Calculate length of output after trimming whitespace
				if (trimmedLen > 0 && trimmedLen < output.Len()) || i > 0 {
					if !bytes.HasSuffix(output.Bytes(), []byte("\n\n")) {
						output.WriteString("\n") // Add newline if one isn't already there
					}
				}
			}
			// Header uses currentIndent for positioning, but fullPathString for the name
			fmt.Fprintf(
				output,
				"%s[[%s]]\n",
				currentIndent,
				fullPathString,
			) // Write the array table header

			// Content uses an increased indent level
			nextIndent := currentIndent + indentUnit // Calculate the next level of indent
			// Recursive call passes the fullPath and nextIndent
			err := formatMap(
				subMap,
				fullPath,
				nextIndent,
				indentUnit,
				output,
			) // Recursively format the submap
			if err != nil {
				// Add context to the error
				return fmt.Errorf(
					"formatting array table '%s' index %d: %w",
					fullPathString,
					i,
					err,
				)
			}
		}
	}
	return nil
}

// formatRegularTables formats and writes regular tables with proper headers and content.
// Regular tables are represented as [section.name] in TOML.
//
// Parameters:
//   - dataMap: Map containing the tables
//   - tableKeys: Slice of keys representing tables
//   - currentPath: Current path to this section
//   - currentIndent: Current indentation string
//   - indentUnit: Unit of indentation for nested content
//   - output: Buffer where formatted output is written
//
// Returns:
//   - error: If any formatting operation fails
func formatRegularTables(
	dataMap map[string]any,
	tableKeys []string,
	currentPath []string, // Path to the parent map
	currentIndent string,
	indentUnit string,
	output *bytes.Buffer,
) error {
	for _, k := range tableKeys {
		// Construct the full path for the table key
		fullPath := append(
			append([]string{}, currentPath...),
			k,
		) // Create a copy and append the new key
		fullPathString := strings.Join(
			fullPath,
			".",
		) // Join the path elements with dots

		subMapInterface := dataMap[k]                  // Get the value associated with the key from the map
		subMap, ok := subMapInterface.(map[string]any) // Assert that the value is a map
		if !ok {
			return fmt.Errorf(
				"internal error: item for table key '%s' is not a map[string]interface{} (got %T)",
				fullPathString, // Use full path in error
				subMapInterface,
			)
		}
		// Add newline separator between sections
		if output.Len() > 0 {
			trimmedLen := len(
				bytes.TrimRight(output.Bytes(), "\n\t "),
			) // Get length after trimming whitespace from end
			if trimmedLen > 0 && trimmedLen < output.Len() {
				if !bytes.HasSuffix(output.Bytes(), []byte("\n\n")) {
					output.WriteString("\n") // Add newline if one isn't already there
				}
			} else if trimmedLen == output.Len() && output.Len() > 0 {
				output.WriteString("\n") // Add newline if the buffer is non empty after trimming
			}
		}
		// Header uses currentIndent for positioning, but fullPathString for the name
		fmt.Fprintf(output, "%s[%s]\n", currentIndent, fullPathString) // Write the table header

		// Content uses an increased indent level
		nextIndent := currentIndent + indentUnit // Calculate the next level of indent
		// Recursive call passes the fullPath and nextIndent
		err := formatMap(
			subMap,
			fullPath,
			nextIndent,
			indentUnit,
			output,
		) // Recursively format the sub-map
		if err != nil {
			// Add context to the error
			return fmt.Errorf("formatting table '%s': %w", fullPathString, err)
		}
	}
	return nil
}

// formatMap is the main recursive function that handles formatting a TOML map.
// It categorizes keys by type, formats them according to TOML conventions,
// and recursively processes nested structures.
//
// Parameters:
//   - dataMap: Map to format
//   - currentPath: Current path of keys leading to this map
//   - currentIndent: Current indentation string
//   - indentUnit: Unit of indentation for nested content
//   - output: Buffer where formatted output is written
//
// Returns:
//   - error: If any formatting operation fails
func formatMap(
	dataMap map[string]any,
	currentPath []string, // Current path of keys leading to this map
	currentIndent string, // Current indentation string for content
	indentUnit string, // Unit of indentation ("" or "  ")
	output *bytes.Buffer,
) error {
	// Get and sort all keys for consistent output
	keys := make(
		[]string,
		0,
		len(dataMap),
	) // Create an empty string slice with a capacity of the map size
	for k := range dataMap {
		keys = append(keys, k) // Add each key from the map to the slice
	}
	sort.Strings(keys) // Sort the slice of keys alphabetically

	maxKeyLen := 0                       // Initialize the maximum key length to 0
	simpleKeys := []string{}             // Slice to store keys of simple key-value pairs
	tableKeys := []string{}              // Slice to store keys of tables
	arrayTableKeys := map[string][]any{} // Map to store keys of array tables and their associated data

	// Categorize keys and find max length for simple keys
	for _, k := range keys {
		v := dataMap[k] // Get the value associated with the key
		if maybeArray, ok := v.([]any); ok &&
			len(maybeArray) > 0 { // Check if it is a non-empty array
			isArrTable := true    // Assume its an array table initially
			containsMaps := false // Flag to track if the array contains map
			// Check if array contains maps (for array tables)
			for _, item := range maybeArray {
				_, itemIsMap := item.(map[string]any) // type assert the item
				if itemIsMap {
					containsMaps = true // set the flag to true
				} else {
					isArrTable = false // If any array entry is not a map, its not an array table
					if containsMaps {  // If we've already found a map
						// Error if array mixes tables and non-tables
						fullPathString := strings.Join(append(append([]string{}, currentPath...), k), ".")
						return fmt.Errorf(
							"key '%s': arrays cannot mix tables and non-tables", fullPathString)
					}
					break
				}
			}
			if isArrTable {
				arrayTableKeys[k] = maybeArray // store the array data
				continue                       // Move to the next key
			}
		}
		// Check if value is a regular table
		if _, ok := v.(map[string]any); ok {
			tableKeys = append(tableKeys, k) // Add the key to the list of table keys
			continue                         // Move to the next key
		}
		// If we get here, it's a simple key-value pair
		simpleKeys = append(simpleKeys, k) // Add the key to the list of simple keys
		if len(k) > maxKeyLen {
			maxKeyLen = len(k) // Track max key length for alignment
		}
	}

	// Format sections in order: simple keys, then array tables, then regular tables
	formatSimpleKeys(dataMap, simpleKeys, maxKeyLen, currentIndent, output)

	// Process array tables
	err := formatArrayTables(arrayTableKeys, currentPath, currentIndent, indentUnit, output)
	if err != nil {
		return err
	}

	// Process regular tables
	err = formatRegularTables(dataMap, tableKeys, currentPath, currentIndent, indentUnit, output)

	// returns err, which will be nil if no error occurred, or the error itself otherwise
	return err
}
