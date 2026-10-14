# prmessage

CLI tool to send templated Slack messages for PRs. Auto-detects project, branch, ticket, PR URL, and Jira ticket title from git context.

## Install

Download binary from [releases](https://github.com/kuudori/prmessage/releases), make executable, and run init:

```bash
chmod +x prmessage-*
mv prmessage-* /usr/local/bin/prmessage
prmessage init
```

### Build from source

```bash
git clone https://github.com/kuudori/prmessage.git
cd prmessage
make build
make install
```

## Usage

```bash
prmessage send        # send message
prmessage send -n     # dry run (preview)
prmessage send -c CH  # override channel
prmessage send -t TKT # override ticket
prmessage send -r TS  # reply in a thread (message ts or pasted permalink)
```

After a successful send, prmessage prints the message permalink.

## Setup

```bash
prmessage init
```

Interactive setup that:

1. Extracts Slack tokens from desktop app automatically (macOS)
2. Asks for default Slack channel
3. Creates empty template to customize

### Requirements

* **git** — project and branch detection
* **[gh](https://cli.github.com/)** — PR info
* **[jira](https://github.com/ankitpokhrel/jira-cli)** — ticket title and URL
* **Slack desktop app** — token extraction (macOS)

## Config

```
~/.config/prmessage/config.yaml   # Slack token + channel
~/.config/prmessage/template.txt  # Message template
```

```yaml
slack:
  token: xoxc-...
  cookie: xoxd-...
  channel: C01DEFAULT        # default channel
channels:                    # optional: per-project overrides (by repo name)
  my-repo: C01TEAMA
  other-repo: "#releases"
protected_branches:          # optional: refuse to send from these (default: main, master)
  - main
  - master
  - release
```

Channel is chosen in order: `-c` flag → `channels` entry for the current project → `slack.channel`.

### Template placeholders

| Placeholder | Source |
|---|---|
| `{project}` | git remote URL |
| `{branch}` | current branch |
| `{ticket}` | extracted from branch name |
| `{ticket_title}` | jira CLI |
| `{ticket_url}` | jira CLI |
| `{pr_url}` | gh CLI |
| `{pr_number}` | gh CLI |

### Example template

```
<{ticket_url}|[{ticket}]> *{ticket_title}*
:github: <{pr_url}|PR #{pr_number}>  ·  {project}
```

## Guards

* Refuses to send on a protected branch (`main`/`master` by default; see `protected_branches`)
* Refuses to send without ticket in branch name
* Refuses to send without open PR

## Commands

| Command | Description |
|---|---|
| `prmessage init` | Set up Slack tokens and config |
| `prmessage send` | Send PR message to Slack |
| `prmessage send -n` | Preview without sending |
| `prmessage update` | Self-update to latest release |
| `prmessage update -f` | Force re-download |
| `prmessage version` | Show version |
| `prmessage help` | Show help |
