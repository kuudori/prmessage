package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"sync"
)

const version = "0.1.1"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		initConfig()
	case "send":
		cmdSend(os.Args[2:])
	case "update":
		cmdUpdate(os.Args[2:])
	case "version":
		fmt.Printf("prmessage %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		errorf("Unknown command: %s", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `prmessage %s — send a templated Slack message for your current PR

Usage:
  prmessage <command> [options]

Commands:
  init       Extract Slack tokens and set up config
  send       Send PR message to Slack
  send -n    Preview message without sending (dry run)
  update     Self-update to latest version
  update -f  Force re-download even if current
  version    Show version
  help       Show this help

Config:    ~/.config/prmessage/config.yaml
Template:  ~/.config/prmessage/template.txt

Get started:
  prmessage init
`, version)
}

func cmdUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	var force bool
	fs.BoolVar(&force, "f", false, "Force update even if already latest")
	fs.BoolVar(&verbose, "v", false, "Verbose output")
	fs.Parse(args)

	info("Checking for updates...")
	latest := checkLatestVersion()
	debug("Current: %s, Latest: %s", version, latest)

	if latest == version && !force {
		info("Already on latest version (%s)", version)
		return
	}

	info("Updating %s → %s", version, latest)
	tmpPath := downloadAsset(latest)
	selfReplace(tmpPath)
	info("Updated to %s", latest)
}

func cmdSend(args []string) {
	fs := flag.NewFlagSet("send", flag.ExitOnError)

	var (
		flagChannel string
		flagTicket  string
		flagProject string
		flagMessage string
		flagDryRun  bool
	)

	fs.StringVar(&flagChannel, "c", "", "Override Slack channel")
	fs.StringVar(&flagTicket, "t", "", "Override ticket number")
	fs.StringVar(&flagProject, "p", "", "Override project name")
	fs.StringVar(&flagMessage, "m", "", "Override message template (one-shot)")
	fs.BoolVar(&flagDryRun, "n", false, "Dry run (preview only)")
	fs.BoolVar(&verbose, "v", false, "Verbose output")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: prmessage send [options]

Options:
  -c CHANNEL    Override Slack channel (name or ID)
  -t TICKET     Override ticket number
  -p PROJECT    Override project name
  -m TEMPLATE   Override message template (one-shot)
  -n            Preview message without sending
  -v            Verbose output

Auto-detects project, branch, ticket, PR URL, and Jira title from git context.
Requires: git, gh (GitHub CLI), jira (Jira CLI)
`)
	}

	fs.Parse(args)

	if gitOutput("rev-parse", "--is-inside-work-tree") != "true" {
		die("Not inside a git repository")
	}

	cfg := loadConfig()

	channel := flagChannel
	if channel == "" {
		channel = cfg.Slack.Channel
	}

	project := flagProject
	if project == "" {
		project = detectProject()
	}
	debug("Project: %s", project)

	branch := detectBranch()
	if branch == "" {
		die("Detached HEAD — cannot determine branch")
	}
	if branch == "main" || branch == "master" {
		die("On %s branch — switch to a feature branch first", branch)
	}
	debug("Branch: %s", branch)

	ticket := flagTicket
	if ticket == "" {
		ticket = extractTicket(branch)
	}
	if ticket == "" {
		die("No ticket found in branch name '%s'", branch)
	}
	debug("Ticket: %s", ticket)

	var (
		prURL      string
		prNumber   int
		ticketData ticketInfo
		wg         sync.WaitGroup
	)
	wg.Add(2)
	go func() { defer wg.Done(); prURL, prNumber = fetchPRInfo(branch) }()
	go func() { defer wg.Done(); ticketData = fetchTicketInfo(ticket) }()
	wg.Wait()

	if prURL == "" {
		die("No open PR for branch '%s' — create a PR first", branch)
	}
	debug("PR: %s (#%d)", prURL, prNumber)
	debug("Ticket: %s — %s", ticketData.URL, ticketData.Title)

	var tmpl string
	if flagMessage != "" {
		tmpl = flagMessage
	} else {
		tmpl = loadTemplate()
	}

	vars := map[string]string{
		"project":      project,
		"branch":       branch,
		"ticket":       ticket,
		"pr_url":       prURL,
		"pr_number":    strconv.Itoa(prNumber),
		"ticket_url":   ticketData.URL,
		"ticket_title": ticketData.Title,
	}

	rendered := renderTemplate(tmpl, vars)

	if flagDryRun {
		fmt.Println()
		bold("--- DRY RUN ---")
		fmt.Printf("Channel: %s\n", channel)
		fmt.Printf("Project: %s\n", project)
		fmt.Printf("Branch:  %s\n", branch)
		fmt.Printf("Ticket:  %s\n", ticket)
		fmt.Println()
		bold("Message:")
		fmt.Println(rendered)
		fmt.Println()
		return
	}

	sendSlackMessage(cfg.Slack, channel, rendered)
}
