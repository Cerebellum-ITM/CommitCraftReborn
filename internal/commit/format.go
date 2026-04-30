package commit

import (
	"errors"
	"fmt"
	"strings"
)

// DefaultTypeFormat is the fallback wrapper applied to a commit tag when
// the user-config TypeFormat is empty. Mirrors the seed value in
// internal/config/types.go.
const DefaultTypeFormat = "[%s]"

// ErrIncompleteCommit is returned by FormatFinalMessage when tag, scope
// or message is empty. The pipeline guarantees all three are populated
// before assembling the final header, so an empty value signals a
// programmer error (e.g. forgetting to run a stage) rather than a
// user-recoverable condition.
var ErrIncompleteCommit = errors.New("commit: tag, scope and message are all required")

// FormatFinalMessage builds the user-visible commit header used by both
// the TUI's post-commit view and the headless `ai` JSON output:
// "<typeFormat(tag)> <scope>: <message>". Returns ErrIncompleteCommit if
// any of tag, scope or message is empty. typeFormat falls back to
// DefaultTypeFormat when empty.
func FormatFinalMessage(typeFormat, tag, scope, message string) (string, error) {
	tag = strings.TrimSpace(tag)
	scope = strings.TrimSpace(scope)
	if tag == "" || scope == "" || message == "" {
		return "", fmt.Errorf("%w (tag=%q scope=%q message-empty=%t)",
			ErrIncompleteCommit, tag, scope, message == "")
	}
	if typeFormat == "" {
		typeFormat = DefaultTypeFormat
	}
	return fmt.Sprintf("%s %s: %s", fmt.Sprintf(typeFormat, tag), scope, message), nil
}
