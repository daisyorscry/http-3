package core

import (
	"path/filepath"
	"strings"
)

// AbsOrEmpty converts relative path to absolute path
// Returns empty string if input is empty
func AbsOrEmpty(p, cwd string) string {
	if strings.TrimSpace(p) == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(cwd, p)
}
