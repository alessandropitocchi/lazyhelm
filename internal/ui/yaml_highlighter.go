package ui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117")). // Azzurro cielo chiaro
			Bold(true)

	stringStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("121")) // Verde menta delicato

	numberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")) // Giallo pastello

	boolStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("219")) // Rosa pastello/lavanda

	commentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("248")) // Grigio chiaro

	nullStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")) // Grigio molto chiaro
)

var (
	commentRegex = regexp.MustCompile(`^\s*#.*$`)
	keyRegex     = regexp.MustCompile(`^(\s*)([a-zA-Z0-9_-]+):\s*(.*)$`)
	numberRegex  = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
	boolRegex    = regexp.MustCompile(`^(true|false|yes|no|on|off)$`)
	nullRegex    = regexp.MustCompile(`^(null|~)$`)
)

func HighlightYAML(line string) string {
	if commentRegex.MatchString(line) {
		return commentStyle.Render(line)
	}

	matches := keyRegex.FindStringSubmatch(line)
	if len(matches) == 4 {
		indent := matches[1]
		key := matches[2]
		value := matches[3]

		result := indent + keyStyle.Render(key+":") + " "

		if value != "" {
			value = strings.TrimSpace(value)
			result += highlightValue(value)
		}

		return result
	}

	return line
}

func highlightValue(value string) string {
	trimmed := strings.Trim(value, `"'`)

	if numberRegex.MatchString(trimmed) {
		return numberStyle.Render(value)
	}

	if boolRegex.MatchString(strings.ToLower(trimmed)) {
		return boolStyle.Render(value)
	}

	if nullRegex.MatchString(trimmed) {
		return nullStyle.Render(value)
	}

	if strings.HasPrefix(value, `"`) || strings.HasPrefix(value, `'`) {
		return stringStyle.Render(value)
	}

	if value != "" && value != "-" && value != "|" && value != ">" {
		return stringStyle.Render(value)
	}

	return value
}

func HighlightYAMLContent(content string) string {
	lines := strings.Split(content, "\n")
	highlighted := make([]string, len(lines))

	for i, line := range lines {
		highlighted[i] = HighlightYAML(line)
	}

	return strings.Join(highlighted, "\n")
}

// HighlightYAMLLine is an alias for HighlightYAML
func HighlightYAMLLine(line string) string {
	return HighlightYAML(line)
}
