package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var ticketRe = regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)

func detectProject() string {
	url := gitOutput("remote", "get-url", "upstream")
	if url == "" {
		url = gitOutput("remote", "get-url", "origin")
	}
	if url != "" {
		base := filepath.Base(url)
		return strings.TrimSuffix(base, ".git")
	}
	toplevel := gitOutput("rev-parse", "--show-toplevel")
	if toplevel != "" {
		return filepath.Base(toplevel)
	}
	return "unknown"
}

func detectBranch() string {
	return gitOutput("branch", "--show-current")
}

func extractTicket(branch string) string {
	return ticketRe.FindString(branch)
}

type prInfo struct {
	URL    string `json:"url"`
	Number int    `json:"number"`
}

func fetchPRInfo() (string, int) {
	if _, err := exec.LookPath("gh"); err != nil {
		warn("gh CLI not found — skipping PR URL")
		return "", 0
	}

	out, err := exec.Command("gh", "pr", "view", "--json", "url,number").Output()
	if err != nil {
		warn("No open PR for current branch")
		return "", 0
	}

	var pr prInfo
	if err := json.Unmarshal(out, &pr); err != nil {
		warn("Failed to parse PR info")
		return "", 0
	}

	return pr.URL, pr.Number
}

func gitOutput(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
