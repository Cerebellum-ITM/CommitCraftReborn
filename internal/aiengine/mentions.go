package aiengine

import "regexp"

// mentionStripRegex matches an `@token` mention preceded by start-of-line
// or whitespace, capturing the leading boundary and the bare token so we
// can drop the `@` while keeping the path text. The path itself is
// information the AI needs ("which files are referenced"); only the
// visual marker is for the human.
var mentionStripRegex = regexp.MustCompile(`(^|\s)@([\w./-]+)`)

// stripMentions removes the leading `@` from every mention in s, leaving
// the file path/identifier intact. Called on every user-supplied snippet
// (key points, summary) right before assembling an AI prompt.
func stripMentions(s string) string {
	return mentionStripRegex.ReplaceAllString(s, "$1$2")
}
