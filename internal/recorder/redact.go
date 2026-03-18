package recorder

import (
	"fmt"
	"os"
	"regexp"
)

type redactPattern struct {
	name     string
	re       *regexp.Regexp
	template string // replacement template; empty means use literal [REDACTED:<name>]
}

var patterns = []redactPattern{
	{"openai-key", regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`), ""},
	{"github-pat", regexp.MustCompile(`ghp_[A-Za-z0-9]{36}`), ""},
	{"aws-key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`), ""},
	{"bearer-token", regexp.MustCompile(`(?i)Bearer\s+\S+`), ""},
	{"generic-secret", regexp.MustCompile(`(?i)((?:^|[\s,;]))(?:password|secret|token|api[_-]?key)\s*[=:]\s*\S+`), "${1}[REDACTED:generic-secret]"},
}

// Redact replaces known secret patterns with [REDACTED:<name>] and reports
// each match to stderr. It never writes secret values to the session log.
func Redact(b []byte) []byte {
	out, _ := RedactWithCount(b)
	return out
}

// RedactWithCount returns the redacted bytes and the number of substitutions made.
func RedactWithCount(b []byte) ([]byte, int) {
	total := 0
	for _, p := range patterns {
		matches := p.re.FindAll(b, -1)
		if len(matches) > 0 {
			total += len(matches)
			fmt.Fprintf(os.Stderr, "[devtrace] redacted: %s pattern matched (%d occurrence(s))\n", p.name, len(matches))
		}
	}
	result := b
	for _, p := range patterns {
		if p.template != "" {
			result = p.re.ReplaceAll(result, []byte(p.template))
		} else {
			result = p.re.ReplaceAllLiteral(result, []byte(fmt.Sprintf("[REDACTED:%s]", p.name)))
		}
	}
	return result, total
}
