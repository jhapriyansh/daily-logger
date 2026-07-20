package main

import (
	"regexp"
	"strings"
)

var slugRe = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func slugify(s string) string {
	s = strings.TrimSpace(s)
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return strings.ToLower(s)
}
