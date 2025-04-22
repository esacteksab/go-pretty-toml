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

// Exported Entry Point
func Format(data map[string]interface{}, indentUnit string, output io.Writer) error {
	var internalBuf bytes.Buffer
	// Start with an empty path for the root map
	err := formatMap(data, []string{}, "", indentUnit, &internalBuf)
	if err != nil {
		return err
	}
	_, err = internalBuf.WriteTo(output)
	return err
}

func formatTomlValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%g", val)
	case bool:
		return strconv.FormatBool(val)
	case time.Time:
		return val.Format(time.RFC3339Nano)
	case nil:
		return "''"
	case []interface{}:
		var elements []string
		for _, item := range val {
			elements = append(elements, formatTomlValue(item))
		}
		return "[" + strings.Join(elements, ", ") + "]"
	default:
		return fmt.Sprintf("<<UNKNOWN TYPE %T>>", v)
	}
}

// formatSimpleKeys now only needs currentIndent, not path
func formatSimpleKeys(
	dataMap map[string]interface{},
	simpleKeys []string,
	maxKeyLen int,
	currentIndent string, // Indent for the line itself
	output *bytes.Buffer,
) {
	for _, k := range simpleKeys {
		v := dataMap[k]
		padding := strings.Repeat(" ", maxKeyLen-len(k))
		formattedValue := formatTomlValue(v)
		fmt.Fprintf(output, "%s%s%s = %s\n", currentIndent, k, padding, formattedValue)
	}
}

// formatArrayTables needs currentPath to build header, currentIndent for position
func formatArrayTables(
	arrayTableKeys map[string][]interface{},
	currentPath []string, // Path to the parent map
	currentIndent string,
	indentUnit string,
	output *bytes.Buffer,
) error {
	sortedArrayTableKeys := make([]string, 0, len(arrayTableKeys))
	for k := range arrayTableKeys {
		sortedArrayTableKeys = append(sortedArrayTableKeys, k)
	}
	sort.Strings(sortedArrayTableKeys)

	for _, k := range sortedArrayTableKeys {
		arrData := arrayTableKeys[k]
		// Construct the full path for the array table key
		fullPath := append(append([]string{}, currentPath...), k) // Create copy before appending
		fullPathString := strings.Join(fullPath, ".")

		for i, item := range arrData {
			subMap, ok := item.(map[string]interface{})
			if !ok {
				return fmt.Errorf(
					"internal error: item in array table '%s' not a map",
					fullPathString,
				)
			}
			// Add newline separator logic...
			if output.Len() > 0 {
				trimmedLen := len(bytes.TrimRight(output.Bytes(), "\n\t "))
				if (trimmedLen > 0 && trimmedLen < output.Len()) || i > 0 {
					if !bytes.HasSuffix(output.Bytes(), []byte("\n\n")) {
						output.WriteString("\n")
					}
				}
			}
			// Header uses currentIndent for positioning, but fullPathString for the name
			fmt.Fprintf(output, "%s[[%s]]\n", currentIndent, fullPathString)

			// Content uses an increased indent level
			nextIndent := currentIndent + indentUnit
			// Recursive call passes the fullPath and nextIndent
			err := formatMap(subMap, fullPath, nextIndent, indentUnit, output)
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

// formatRegularTables needs currentPath to build header, currentIndent for position
func formatRegularTables(
	dataMap map[string]interface{},
	tableKeys []string,
	currentPath []string, // Path to the parent map
	currentIndent string,
	indentUnit string,
	output *bytes.Buffer,
) error {
	for _, k := range tableKeys {
		// Construct the full path for the table key
		fullPath := append(append([]string{}, currentPath...), k) // Create copy before appending
		fullPathString := strings.Join(fullPath, ".")

		subMapInterface := dataMap[k]
		subMap, ok := subMapInterface.(map[string]interface{})
		if !ok {
			return fmt.Errorf(
				"internal error: item for table key '%s' is not a map[string]interface{} (got %T)",
				fullPathString, // Use full path in error
				subMapInterface,
			)
		}
		// Add newline separator logic...
		if output.Len() > 0 {
			trimmedLen := len(bytes.TrimRight(output.Bytes(), "\n\t "))
			if trimmedLen > 0 && trimmedLen < output.Len() {
				if !bytes.HasSuffix(output.Bytes(), []byte("\n\n")) {
					output.WriteString("\n")
				}
			} else if trimmedLen == output.Len() && output.Len() > 0 {
				output.WriteString("\n")
			}
		}
		// Header uses currentIndent for positioning, but fullPathString for the name
		fmt.Fprintf(output, "%s[%s]\n", currentIndent, fullPathString)

		// Content uses an increased indent level
		nextIndent := currentIndent + indentUnit
		// Recursive call passes the fullPath and nextIndent
		err := formatMap(subMap, fullPath, nextIndent, indentUnit, output)
		if err != nil {
			// Add context to the error
			return fmt.Errorf("formatting table '%s': %w", fullPathString, err)
		}
	}
	return nil
}

// Main Recursive Formatting Function
func formatMap(
	dataMap map[string]interface{},
	currentPath []string, // Current path of keys leading to this map
	currentIndent string, // Current indentation string for content
	indentUnit string, // Unit of indentation ("" or "  ")
	output *bytes.Buffer,
) error {
	keys := make([]string, 0, len(dataMap))
	for k := range dataMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	maxKeyLen := 0
	simpleKeys := []string{}
	tableKeys := []string{}
	arrayTableKeys := map[string][]interface{}{}

	// Categorize keys and find max length for simple keys
	for _, k := range keys {
		v := dataMap[k]
		if maybeArray, ok := v.([]interface{}); ok && len(maybeArray) > 0 {
			isArrTable := true
			containsMaps := false
			for _, item := range maybeArray {
				_, itemIsMap := item.(map[string]interface{})
				if itemIsMap {
					containsMaps = true
				} else {
					isArrTable = false
					if containsMaps {
						// Generate error using the key 'k' relative to the current path
						fullPathString := strings.Join(append(append([]string{}, currentPath...), k), ".")
						return fmt.Errorf(
							"key '%s': arrays cannot mix tables and non-tables", fullPathString)
					}
					break
				}
			}
			if isArrTable {
				arrayTableKeys[k] = maybeArray
				continue
			}
		}
		if _, ok := v.(map[string]interface{}); ok {
			tableKeys = append(tableKeys, k)
			continue
		}
		simpleKeys = append(simpleKeys, k)
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}

	// Format sections using helpers
	// Simple keys are printed with currentIndent
	formatSimpleKeys(dataMap, simpleKeys, maxKeyLen, currentIndent, output)

	// Array tables need the current path to build headers
	err := formatArrayTables(arrayTableKeys, currentPath, currentIndent, indentUnit, output)
	if err != nil {
		return err
	}

	// Regular tables need the current path to build headers
	err = formatRegularTables(dataMap, tableKeys, currentPath, currentIndent, indentUnit, output)
	if err != nil {
		return err
	}

	return nil
}
