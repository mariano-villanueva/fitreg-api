package handlers

import (
	"strconv"
	"strings"
)

// extractID parses the numeric ID from a URL path given a prefix.
func extractID(path, prefix string) (int64, error) {
	s := strings.TrimPrefix(path, prefix)
	if idx := strings.Index(s, "/"); idx != -1 {
		s = s[:idx]
	}
	return strconv.ParseInt(s, 10, 64)
}

func truncateDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
