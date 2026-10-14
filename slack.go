package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	slackPostURL      = "https://slack.com/api/chat.postMessage"
	slackPermalinkURL = "https://slack.com/api/chat.getPermalink"
)

// Slack archive links end in /pXXXXXXXXXXXXXXXXX — the message ts with its
// decimal point removed. parseThreadTS also accepts a raw ts unchanged.
var archiveTSRe = regexp.MustCompile(`/p(\d{10})(\d{6})`)

type slackPayload struct {
	Channel     string `json:"channel"`
	Text        string `json:"text"`
	ThreadTS    string `json:"thread_ts,omitempty"`
	UnfurlLinks bool   `json:"unfurl_links"`
	UnfurlMedia bool   `json:"unfurl_media"`
}

// parseThreadTS normalizes a thread reference, accepting either a raw Slack ts
// ("1700000000.123456") or a message permalink pasted from the app.
func parseThreadTS(ref string) string {
	ref = strings.TrimSpace(ref)
	if m := archiveTSRe.FindStringSubmatch(ref); m != nil {
		return m[1] + "." + m[2]
	}
	return ref
}

func setSlackAuth(req *http.Request, slack SlackConfig) {
	req.Header.Set("Authorization", "Bearer "+slack.Token)
	if strings.HasPrefix(slack.Token, "xoxc-") && slack.Cookie != "" {
		req.Header.Set("Cookie", "d="+slack.Cookie)
	}
}

func sendSlackMessage(slack SlackConfig, channel, text, threadTS string) {
	payload := slackPayload{
		Channel:     channel,
		Text:        text,
		ThreadTS:    threadTS,
		UnfurlLinks: false,
		UnfurlMedia: false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		die("Failed to marshal Slack payload: %v", err)
	}

	debug("Payload: %s", string(body))

	req, err := http.NewRequest("POST", slackPostURL, bytes.NewReader(body))
	if err != nil {
		die("Failed to create Slack request: %v", err)
	}

	setSlackAuth(req, slack)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		die("Slack API request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	debug("Response: %s", string(respBody))

	var result struct {
		OK      bool   `json:"ok"`
		Channel string `json:"channel"`
		TS      string `json:"ts"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		die("Failed to parse Slack response: %v", err)
	}

	if !result.OK {
		die("Slack API error: %s", result.Error)
	}

	info("Message sent to %s", channel)
	if link := fetchPermalink(slack, result.Channel, result.TS); link != "" {
		info("Permalink: %s", link)
	}
}

// fetchPermalink resolves the archive link for a just-posted message. Any
// failure is non-fatal — the message is already sent, so we just skip the link.
func fetchPermalink(slack SlackConfig, channel, ts string) string {
	if channel == "" || ts == "" {
		return ""
	}

	q := url.Values{"channel": {channel}, "message_ts": {ts}}
	req, err := http.NewRequest("GET", slackPermalinkURL+"?"+q.Encode(), nil)
	if err != nil {
		return ""
	}
	setSlackAuth(req, slack)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		OK        bool   `json:"ok"`
		Permalink string `json:"permalink"`
	}
	if json.NewDecoder(resp.Body).Decode(&result) != nil || !result.OK {
		return ""
	}
	return result.Permalink
}
