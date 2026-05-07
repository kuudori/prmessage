package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

const slackAPIURL = "https://slack.com/api/chat.postMessage"

type slackPayload struct {
	Channel     string `json:"channel"`
	Text        string `json:"text"`
	UnfurlLinks bool   `json:"unfurl_links"`
	UnfurlMedia bool   `json:"unfurl_media"`
}

func sendSlackMessage(slack SlackConfig, channel, text string) {
	payload := slackPayload{
		Channel:     channel,
		Text:        text,
		UnfurlLinks: false,
		UnfurlMedia: false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		die("Failed to marshal Slack payload: %v", err)
	}

	debug("Payload: %s", string(body))

	req, err := http.NewRequest("POST", slackAPIURL, bytes.NewReader(body))
	if err != nil {
		die("Failed to create Slack request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+slack.Token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	if strings.HasPrefix(slack.Token, "xoxc-") && slack.Cookie != "" {
		req.Header.Set("Cookie", "d="+slack.Cookie)
	}

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
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		die("Failed to parse Slack response: %v", err)
	}

	if !result.OK {
		die("Slack API error: %s", result.Error)
	}

	info("Message sent to %s", channel)
}
