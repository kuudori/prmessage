package main

import (
	"os"
	"regexp"
	"strings"
)

var emptyLinkRe = regexp.MustCompile(`<\|[^>]*>`)

func loadTemplate() string {
	path := templatePath()

	data, err := os.ReadFile(path)
	if err != nil {
		die("Template not found at %s\nRun 'prmessage init' to create it.", path)
	}

	tpl := strings.TrimSpace(string(data))
	if tpl == "" {
		die("Template file is empty: %s", path)
	}

	return tpl
}

func renderTemplate(tmpl string, vars map[string]string) string {
	result := tmpl
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}

	result = emptyLinkRe.ReplaceAllString(result, "(no link)")

	return result
}
