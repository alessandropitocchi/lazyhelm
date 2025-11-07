// Copyright 2025 Alessandro Pitocchi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui

import (
	"strings"
)

func GetYAMLPath(lines []string, lineNum int) string {
	if lineNum < 0 || lineNum >= len(lines) {
		return ""
	}

	currentLine := lines[lineNum]
	trimmedLine := strings.TrimSpace(currentLine)

	// Skip empty lines and comments
	if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
		return ""
	}

	currentIndent := getIndentLevel(currentLine)
	currentField := extractKey(currentLine)

	// If current line doesn't have a key, look backwards for the nearest parent key
	if currentField == "" {
		// Check if this is a list item (starts with -)
		isListItem := strings.HasPrefix(trimmedLine, "-")

		for i := lineNum - 1; i >= 0; i-- {
			line := lines[i]
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}

			indent := getIndentLevel(line)

			// For list items, look for parent at lower indent
			// For other values, look at same or lower indent
			targetIndent := currentIndent
			if isListItem {
				targetIndent = currentIndent - 2 // List items are typically 2 spaces indented from parent
				if targetIndent < 0 {
					targetIndent = 0
				}
			}

			// Found a key at appropriate indent level
			if indent < currentIndent || (indent == targetIndent && extractKey(line) != "") {
				field := extractKey(line)
				if field != "" {
					// Use this as our starting point
					lineNum = i
					currentLine = line
					currentIndent = indent
					currentField = field
					break
				}
			}
		}

		// Still no field found
		if currentField == "" {
			return ""
		}
	}

	path := []string{}

	// Build the path by walking backwards through parent keys
	for i := lineNum - 1; i >= 0; i-- {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := getIndentLevel(line)

		if indent < currentIndent {
			field := extractKey(line)
			if field != "" {
				path = append([]string{field}, path...)
				currentIndent = indent
			}
		}

		if indent == 0 {
			break
		}
	}

	path = append(path, currentField)
	return strings.Join(path, ".")
}

func getIndentLevel(line string) int {
	count := 0
	for _, ch := range line {
		if ch == ' ' {
			count++
		} else if ch == '\t' {
			count += 2
		} else {
			break
		}
	}
	return count
}

func extractKey(line string) string {
	trimmed := strings.TrimSpace(line)
	if idx := strings.Index(trimmed, ":"); idx > 0 {
		return trimmed[:idx]
	}
	return ""
}

type DiffLine struct {
	Type    string // "added", "removed", "unchanged", "modified"
	Line    string
	LineNum int
}

func DiffYAML(oldContent, newContent string) []DiffLine {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	oldMap := make(map[string]struct {
		line   string
		lineNum int
	})
	newMap := make(map[string]struct {
		line   string
		lineNum int
	})

	// Build maps with line numbers
	for i, line := range oldLines {
		key := extractKey(line)
		if key != "" {
			oldMap[key] = struct {
				line   string
				lineNum int
			}{line, i}
		}
	}

	for i, line := range newLines {
		key := extractKey(line)
		if key != "" {
			newMap[key] = struct {
				line   string
				lineNum int
			}{line, i}
		}
	}

	result := make([]DiffLine, 0)
	contextLines := 2 // Number of context lines to show around changes

	// Track which lines are changes or near changes
	isChange := make(map[int]bool)

	// Find all changes first
	for key, newData := range newMap {
		if oldData, exists := oldMap[key]; exists {
			if oldData.line != newData.line {
				// Modified line - mark it and add both old and new
				isChange[newData.lineNum] = true
			}
		} else {
			// Added line
			isChange[newData.lineNum] = true
		}
	}

	// Find removed lines
	removedKeys := make([]string, 0)
	for key := range oldMap {
		if _, exists := newMap[key]; !exists {
			removedKeys = append(removedKeys, key)
		}
	}

	// Build result with changes and context
	for i, newLine := range newLines {
		key := extractKey(newLine)

		// Check if this line or nearby lines are changes
		hasNearbyChange := false
		for j := i - contextLines; j <= i + contextLines; j++ {
			if isChange[j] {
				hasNearbyChange = true
				break
			}
		}

		if !hasNearbyChange {
			continue // Skip lines far from changes
		}

		if key != "" {
			if oldData, exists := oldMap[key]; exists {
				if oldData.line != newLine {
					// Show old line first, then new line
					result = append(result, DiffLine{Type: "removed", Line: oldData.line, LineNum: oldData.lineNum})
					result = append(result, DiffLine{Type: "added", Line: newLine, LineNum: i})
				} else {
					// Context line (unchanged)
					result = append(result, DiffLine{Type: "unchanged", Line: newLine, LineNum: i})
				}
			} else {
				// Added line
				result = append(result, DiffLine{Type: "added", Line: newLine, LineNum: i})
			}
		} else {
			// Context line (empty or comment)
			result = append(result, DiffLine{Type: "unchanged", Line: newLine, LineNum: i})
		}
	}

	// Add removed lines at the end with context
	for _, key := range removedKeys {
		oldData := oldMap[key]
		result = append(result, DiffLine{Type: "removed", Line: oldData.line, LineNum: oldData.lineNum})
	}

	return result
}
