package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var ticketRe = regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)

func remoteURL() string {
	url := gitOutput("remote", "get-url", "upstream")
	if url == "" {
		url = gitOutput("remote", "get-url", "origin")
	}
	return url
}

func detectProject() string {
	if url := remoteURL(); url != "" {
		return strings.TrimSuffix(filepath.Base(url), ".git")
	}
	if toplevel := gitOutput("rev-parse", "--show-toplevel"); toplevel != "" {
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

func parseRepoSlug(url string) string {
	// SSH: git@github.com:owner/repo.git
	if strings.Contains(url, ":") && !strings.Contains(url, "://") {
		parts := strings.SplitN(url, ":", 2)
		if len(parts) == 2 {
			return strings.TrimSuffix(parts[1], ".git")
		}
	}
	// HTTPS: https://github.com/owner/repo.git
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return ""
}

func fetchPRInfo(branch string) (string, int) {
	if _, err := exec.LookPath("gh"); err != nil {
		warn("gh CLI not found — skipping PR URL")
		return "", 0
	}

	out, err := exec.Command("gh", "pr", "view", "--json", "url,number").Output()
	if err != nil {
		// Fallback: gh pr list with explicit repo + branch
		// Handles worktrees and cross-repository (fork) PRs
		repo := parseRepoSlug(remoteURL())
		if repo == "" || branch == "" {
			warn("No open PR for current branch")
			return "", 0
		}
		debug("Fallback: gh pr list -R %s --head %s", repo, branch)
		out, err = exec.Command("gh", "pr", "list", "-R", repo, "--head", branch, "--json", "url,number", "--limit", "1").Output()
		if err != nil {
			warn("No open PR for current branch")
			return "", 0
		}
	}

	// gh pr view returns object, gh pr list returns array
	out = normalizeGHOutput(out)

	var pr prInfo
	if err := json.Unmarshal(out, &pr); err != nil {
		warn("Failed to parse PR info")
		return "", 0
	}

	return pr.URL, pr.Number
}

func normalizeGHOutput(data []byte) []byte {
	data = []byte(strings.TrimSpace(string(data)))
	if len(data) > 0 && data[0] == '[' {
		var arr []json.RawMessage
		if json.Unmarshal(data, &arr) == nil && len(arr) > 0 {
			return arr[0]
		}
		return []byte("{}")
	}
	return data
}

func gitOutput(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
