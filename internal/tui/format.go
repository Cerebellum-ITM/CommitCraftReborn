package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ExtractJSONError pulls a JSON-shaped error message out of an upstream API
// response so the user sees something more useful than the default fallback.
func ExtractJSONError(errText string) string {
	var defaultError string = "There was a problem with the AI request"

	re := regexp.MustCompile(`{.*"message".*}`)
	match := re.FindString(
		errText,
	)
	if match != "" {
		return match
	}

	return defaultError
}

// TruncatePath shortens a path to the last `levels` components, prefixing
// with "..." when something was dropped. "/a/b/c/d" with levels=2 →
// ".../c/d".
func TruncatePath(path string, levels int) string {
	if levels <= 0 || path == "" {
		return ""
	}

	parts := strings.Split(path, string(os.PathSeparator))
	filteredParts := []string{}
	for _, part := range parts {
		if part != "" {
			filteredParts = append(filteredParts, part)
		}
	}

	if len(filteredParts) <= levels {
		return path
	}

	startIndex := len(filteredParts) - levels
	truncatedParts := filteredParts[startIndex:]

	prefix := ""
	if startIndex > 0 {
		prefix = "..." + string(os.PathSeparator)
	}

	return prefix + strings.Join(truncatedParts, string(os.PathSeparator))
}

// TruncateString returns the first line of s, capped at maxLen runes,
// appending "..." when truncated.
func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	firstLine := s
	if newlineIndex := strings.Index(s, "\n"); newlineIndex != -1 {
		firstLine = s[:newlineIndex]
	}

	runes := []rune(firstLine)
	if len(runes) <= maxLen {
		return firstLine
	}

	if maxLen < 3 {
		return string(runes[:maxLen])
	}

	return string(runes[:maxLen-3]) + "..."
}

// TruncateMessageLines applies TruncateString to every line of message,
// preserving blank lines.
func TruncateMessageLines(message string, width int) string {
	lines := strings.Split(message, "\n")
	var formattedLines []string

	for _, line := range lines {
		if line == "" {
			formattedLines = append(formattedLines, "")
			continue
		}
		formattedLines = append(formattedLines, TruncateString(line, width))
	}

	return strings.Join(formattedLines, "\n")
}

// GetNerdFontIcon picks a Nerd Font glyph for a filename based on its
// extension, with a few special-cased basenames.
func GetNerdFontIcon(filename string, isDir bool) string {
	if isDir {
		return ""
	}

	extension := strings.ToLower(filepath.Ext(filename))
	name := strings.ToLower(filename)

	switch extension {
	case ".go":
		return "󰟓"
	case ".py":
		return ""
	case ".js":
		return "󰌞"
	case ".ts":
		return "󰛦"
	case ".java":
		return ""
	case ".cs":
		return "󰌛"
	case ".rs":
		return ""
	case ".c":
		return "󰙱"
	case ".cpp", ".h":
		return "󰙲"
	case ".json":
		return "󰘦"
	case ".yml", ".yaml":
		return ""
	case ".xml":
		return "󰗀"
	case ".toml":
		return ""
	case ".env":
		return ""
	case ".md", ".mdx":
		return ""
	case ".git":
		return " Git"
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return ""
	case ".zip", ".tar", ".gz", ".rar":
		return "󰿺"
	default:
		switch name {
		case "docker-compose.yml", "dockerfile":
			return "󰡨"
		case "makefile":
			return ""
		case "readme.md":
			return "󰂺"
		case "license":
			return "󰿃"
		case ".gitignore":
			return ""
		}
		return ""
	}
}

// BumpVersionPatch increments the last numeric segment of a version string
// (e.g. "v0.6.1" → "v0.6.2"). Non-numeric trailing characters are preserved.
// Returns "" if the input has no digits at all.
func BumpVersionPatch(v string) string {
	if v == "" {
		return ""
	}
	runes := []rune(v)
	end := len(runes)
	for end > 0 && !(runes[end-1] >= '0' && runes[end-1] <= '9') {
		end--
	}
	if end == 0 {
		return ""
	}
	start := end
	for start > 0 && runes[start-1] >= '0' && runes[start-1] <= '9' {
		start--
	}
	num := 0
	for i := start; i < end; i++ {
		num = num*10 + int(runes[i]-'0')
	}
	num++
	out := string(runes[:start]) + fmt.Sprintf("%d", num) + string(runes[end:])
	return out
}
