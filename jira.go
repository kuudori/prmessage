package main

import (
	"encoding/json"
	"net/url"
	"os/exec"
)

type ticketInfo struct {
	Title string
	URL   string
}

func fetchTicketInfo(ticket string) ticketInfo {
	if _, err := exec.LookPath("jira"); err != nil {
		warn("jira CLI not found — skipping ticket info")
		return ticketInfo{}
	}

	out, err := exec.Command("jira", "issue", "view", ticket, "--raw").Output()
	if err != nil {
		warn("jira issue view failed for %s", ticket)
		return ticketInfo{}
	}

	var result struct {
		Self   string `json:"self"`
		Fields struct {
			Summary string `json:"summary"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		warn("Failed to parse jira output for %s", ticket)
		return ticketInfo{}
	}

	browseURL := ""
	if result.Self != "" {
		if u, err := url.Parse(result.Self); err == nil {
			browseURL = u.Scheme + "://" + u.Host + "/browse/" + ticket
		}
	}

	return ticketInfo{
		Title: result.Fields.Summary,
		URL:   browseURL,
	}
}
