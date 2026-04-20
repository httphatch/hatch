package session

import (
	"fmt"
	"regexp"
	"strings"
)

// templatePattern matches {{port}} and {{port:service_name}}.
var templatePattern = regexp.MustCompile(`\{\{port(?::([a-zA-Z0-9_-]+))?\}\}`)

// SubstituteCommand replaces template variables in a command string.
// {{port}} resolves to the port for the current service.
// {{port:web}} resolves to the port for the named service.
func SubstituteCommand(cmd, serviceName string, ports map[string]int) string {
	return templatePattern.ReplaceAllStringFunc(cmd, func(match string) string {
		subs := templatePattern.FindStringSubmatch(match)
		if subs[1] != "" {
			if p, ok := ports[subs[1]]; ok {
				return fmt.Sprintf("%d", p)
			}
			return match // leave unresolved if service not found
		}
		if p, ok := ports[serviceName]; ok {
			return fmt.Sprintf("%d", p)
		}
		return match
	})
}

// SubstituteEnv replaces template variables in environment variable values.
// Only the value portion (after =) is substituted.
func SubstituteEnv(env []string, serviceName string, ports map[string]int) []string {
	result := make([]string, len(env))
	for i, entry := range env {
		if k, v, ok := strings.Cut(entry, "="); ok {
			result[i] = k + "=" + SubstituteCommand(v, serviceName, ports)
		} else {
			result[i] = entry
		}
	}
	return result
}

// HasPortTemplate returns true if the string contains any {{port}} template.
func HasPortTemplate(s string) bool {
	return templatePattern.MatchString(s)
}
