package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const configDirName = ".config/prmessage"

type Config struct {
	Slack SlackConfig `yaml:"slack"`
}

type SlackConfig struct {
	Token   string `yaml:"token"`
	Cookie  string `yaml:"cookie,omitempty"`
	Channel string `yaml:"channel"`
}

func configDir() string {
	return filepath.Join(mustHomeDir(), configDirName)
}

func configPath() string {
	return filepath.Join(configDir(), "config.yaml")
}

func templatePath() string {
	return filepath.Join(configDir(), "template.txt")
}

func loadConfig() *Config {
	path := configPath()

	data, err := os.ReadFile(path)
	if err != nil {
		die("Config not found at %s\nRun 'prmessage init' to create it.", path)
	}

	checkPermissions(path)

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		die("Failed to parse config: %v", err)
	}

	if cfg.Slack.Token == "" {
		die("No Slack token in config. Run 'prmessage init' to set up.")
	}
	if cfg.Slack.Channel == "" {
		die("No Slack channel in config. Run 'prmessage init' to set up.")
	}

	return &cfg
}

func checkPermissions(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	mode := info.Mode().Perm()
	if mode&0o077 != 0 {
		warn("Config %s has permissions %o, should be 600", path, mode)
	}
}

func initConfig() {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		die("Failed to create config dir: %v", err)
	}

	path := configPath()
	if _, err := os.Stat(path); err == nil {
		ans := prompt("Config exists at %s. Overwrite? [y/N] ", path)
		if !strings.HasPrefix(strings.ToLower(ans), "y") {
			info("Aborted.")
			os.Exit(0)
		}
	}

	fmt.Println()
	bold("prmessage setup")
	fmt.Println()

	var token, cookie string

	fmt.Println("  Close Slack desktop app before continuing.")
	fmt.Println()
	prompt("Press Enter when Slack is closed...")
	fmt.Println()
	info("Extracting Slack tokens from desktop app...")
	token, cookie, err := extractSlackTokens()
	if err != nil {
		warn("Auto-extraction failed: %v", err)
		fmt.Println()
		fmt.Println("  Grab tokens manually from browser DevTools:")
		fmt.Println("  1. Open Slack in browser → DevTools → Network tab")
		fmt.Println("  2. Do any action in Slack (switch channel, etc)")
		fmt.Println("  3. Find request to api.slack.com")
		fmt.Println("  4. Request payload → copy 'token' (xoxc-...)")
		fmt.Println("  5. Request headers → Cookie → copy 'd=xoxd-...'")
		fmt.Println()

		token = prompt("Slack token (xoxc-...): ")
		if strings.HasPrefix(token, "xoxc-") {
			cookie = prompt("Slack cookie (xoxd-...): ")
		}
	} else {
		info("Token: %s...%s", token[:20], token[len(token)-6:])
		if cookie != "" {
			info("Cookie: %s...%s", cookie[:20], cookie[len(cookie)-6:])
		}
	}

	fmt.Println("  Tip: right-click a channel name in Slack → Copy → Copy link")
	fmt.Println("       The channel ID is the last part of the URL")
	fmt.Println()
	channel := prompt("Default Slack channel (ID or #name): ")

	cfg := Config{
		Slack: SlackConfig{
			Token:   token,
			Cookie:  cookie,
			Channel: channel,
		},
	}

	saveConfig(&cfg)

	tplPath := templatePath()
	if _, err := os.Stat(tplPath); os.IsNotExist(err) {
		if err := os.WriteFile(tplPath, []byte(""), 0o644); err != nil {
			die("Failed to write template: %v", err)
		}
		info("Template created at %s (empty — edit before sending)", tplPath)
	}

	fmt.Println()
	bold("Setup complete!")
	fmt.Println()
	fmt.Println("  Files created:")
	fmt.Printf("    Config:   %s\n", configPath())
	fmt.Printf("    Template: %s\n", tplPath)
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Println("    1. Edit template to match your team's format:")
	fmt.Printf("       $EDITOR %s\n", tplPath)
	fmt.Println("    2. Switch to a feature branch and test:")
	fmt.Println("       prmessage send -n")
	fmt.Println("    3. Send for real:")
	fmt.Println("       prmessage send")
	fmt.Println()
	fmt.Println("  Placeholders available in template:")
	fmt.Println("    {project}  {branch}  {ticket}  {ticket_title}")
	fmt.Println("    {ticket_url}  {pr_url}  {pr_number}")

	if _, err := exec.LookPath("prmessage"); err != nil {
		binDir := filepath.Dir(os.Args[0])
		fmt.Println()
		fmt.Println("  Add prmessage to your PATH:")
		fmt.Printf("    export PATH=\"%s:$PATH\"\n", binDir)
		fmt.Println("    # Add the line above to ~/.zshrc to make it permanent")
	}
	fmt.Println()
}

func saveConfig(cfg *Config) {
	path := configPath()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		die("Failed to marshal config: %v", err)
	}

	header := "# prmessage config\n\n"
	if err := os.WriteFile(path, append([]byte(header), data...), 0o600); err != nil {
		die("Failed to write config: %v", err)
	}
	info("Config written to %s (permissions: 600)", path)
}

func prompt(format string, args ...any) string {
	fmt.Printf(format, args...)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}
