package process

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadEnvFile reads a .env file and returns a map of key-value pairs.
// It supports comments (#), empty lines, quoted values (single and double),
// and inline comments after unquoted values.
func LoadEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open env file: %w", err)
	}
	defer func() { _ = f.Close() }()

	env := make(map[string]string)
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, err := parseEnvLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		env[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read env file: %w", err)
	}

	return env, nil
}

func parseEnvLine(line string) (string, string, error) {
	// Split on first '='.
	idx := strings.IndexByte(line, '=')
	if idx < 0 {
		return "", "", fmt.Errorf("invalid line: no '=' found")
	}

	key := strings.TrimSpace(line[:idx])
	if key == "" {
		return "", "", fmt.Errorf("invalid line: empty key")
	}

	raw := strings.TrimSpace(line[idx+1:])

	// Handle quoted values.
	if len(raw) >= 2 {
		if (raw[0] == '"' && raw[len(raw)-1] == '"') ||
			(raw[0] == '\'' && raw[len(raw)-1] == '\'') {
			return key, raw[1 : len(raw)-1], nil
		}
	}

	// Strip inline comment for unquoted values.
	if ci := strings.IndexByte(raw, '#'); ci >= 0 {
		raw = strings.TrimSpace(raw[:ci])
	}

	return key, raw, nil
}
