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

// Format takes parsed TOML data and formats it according to the rules,
// writing the result to the provided io.Writer.
func Format(data map[string]interface{}, indentUnit string, output io.Writer) error {
	// Use an internal buffer for the recursive formatting functions
	var internalBuf bytes.Buffer
	err := formatMap(data, "", indentUnit, &internalBuf) // Call the unexported recursive function
	if err != nil {
		return err // Propagate errors from formatting
	}
	// Write the final formatted result from the buffer to the provided writer
	_, err = internalBuf.WriteTo(output)
	return err // Return any error from writing the output
}

// Helper: formatTomlValue
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
		return "''" // Placeholder for nil
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

// formatMap Helpers

func formatSimpleKeys(
	dataMap map[string]interface{},
	simpleKeys []string,
	maxKeyLen int,
	currentIndent string,
	output *bytes.Buffer,
) {
	for _, k := range simpleKeys {
		v := dataMap[k]
		padding := strings.Repeat(" ", maxKeyLen-len(k))
		formattedValue := formatTomlValue(v)
		fmt.Fprintf(output, "%s%s%s = %s\n", currentIndent, k, padding, formattedValue)
	}
}

func formatArrayTables(
	arrayTableKeys map[string][]interface{},
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
		for i, item := range arrData {
			subMap, ok := item.(map[string]interface{})
			if !ok {
				return fmt.Errorf("internal error: item in array table '%s' not a map", k)
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
			fmt.Fprintf(output, "%s[[%s]]\n", currentIndent, k)
			nextIndent := currentIndent + indentUnit
			err := formatMap(subMap, nextIndent, indentUnit, output)
			if err != nil {
				return fmt.Errorf("formatting array table '%s[%d]': %w", k, i, err)
			}
		}
	}
	return nil
}

func formatRegularTables(
	dataMap map[string]interface{},
	tableKeys []string,
	currentIndent string,
	indentUnit string,
	output *bytes.Buffer,
) error {
	for _, k := range tableKeys {
		subMapInterface := dataMap[k]
		subMap, ok := subMapInterface.(map[string]interface{})
		if !ok {
			return fmt.Errorf(
				"internal error: item for table key '%s' is not a map[string]interface{} (got %T)",
				k,
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
		fmt.Fprintf(output, "%s[%s]\n", currentIndent, k)
		nextIndent := currentIndent + indentUnit
		err := formatMap(subMap, nextIndent, indentUnit, output)
		if err != nil {
			return fmt.Errorf("formatting table '%s': %w", k, err)
		}
	}
	return nil
}

// Main Recursive Formatting Function
func formatMap(
	dataMap map[string]interface{},
	currentIndent string,
	indentUnit string,
	output *bytes.Buffer, // Stays as *bytes.Buffer for internal use
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

	// Pass 1: Categorize keys and find max length for simple keys
	for _, k := range keys {
		v := dataMap[k]

		//  Check for Array of Tables FIRST
		if maybeArray, ok := v.([]interface{}); ok && len(maybeArray) > 0 {
			isArrTable := true
			containsMaps := false // Track if we saw *any* maps
			for _, item := range maybeArray {
				_, itemIsMap := item.(map[string]interface{})
				if itemIsMap {
					containsMaps = true
				} else {
					// Found a non-map item
					isArrTable = false
					if containsMaps {
						return fmt.Errorf(
							"key '%s': arrays cannot mix tables and non-tables", k)
					}
					// If no maps were seen yet, break and let it be treated as a simple array later
					break
				}
			}
			// If the loop finished and it's purely an array of tables:
			if isArrTable {
				arrayTableKeys[k] = maybeArray
				continue // Successfully categorized as array table
			}
		}

		// Check for Regular Table SECOND
		if _, ok := v.(map[string]interface{}); ok {
			tableKeys = append(tableKeys, k)
			continue // Successfully categorized as regular table
		}

		//  Otherwise, treat as Simple Key (includes simple arrays)
		simpleKeys = append(simpleKeys, k)
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	} // End of Pass 1 loop

	// Pass 2: Format sections using helpers
	formatSimpleKeys(dataMap, simpleKeys, maxKeyLen, currentIndent, output)

	err := formatArrayTables(arrayTableKeys, currentIndent, indentUnit, output)
	if err != nil {
		return err
	}

	err = formatRegularTables(dataMap, tableKeys, currentIndent, indentUnit, output)
	if err != nil {
		return err
	}

	return nil
}
