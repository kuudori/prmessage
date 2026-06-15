package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	repoOwner = "kuudori"
	repoName  = "prmessage"
)

func checkLatestVersion() string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		die("Failed to check for updates: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		die("GitHub API returned %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		die("Failed to parse release info: %v", err)
	}

	return strings.TrimPrefix(release.TagName, "v")
}

func downloadAsset(tag string) string {
	asset := fmt.Sprintf("prmessage-%s-%s", runtime.GOOS, runtime.GOARCH)
	url := fmt.Sprintf("https://github.com/%s/%s/releases/download/v%s/%s", repoOwner, repoName, tag, asset)

	debug("Downloading %s", url)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		die("Download failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		die("Asset not found: %s (HTTP %d)", asset, resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "prmessage-update-*")
	if err != nil {
		die("Failed to create temp file: %v", err)
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		die("Download failed: %v", err)
	}
	tmp.Close()

	return tmp.Name()
}

func selfReplace(tmpPath string) {
	exe, err := os.Executable()
	if err != nil {
		die("Cannot determine executable path: %v", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		die("Cannot resolve executable path: %v", err)
	}

	if err := os.Chmod(tmpPath, 0o755); err != nil {
		die("Failed to set permissions: %v", err)
	}

	// Atomic replace: rename within same directory
	dir := filepath.Dir(exe)
	staged := filepath.Join(dir, ".prmessage-update")

	if err := os.Rename(tmpPath, staged); err != nil {
		// Cross-device fallback: copy
		copyFile(tmpPath, staged)
		os.Remove(tmpPath)
	}

	if err := os.Rename(staged, exe); err != nil {
		die("Failed to replace binary: %v", err)
	}
}

func copyFile(src, dst string) {
	in, err := os.Open(src)
	if err != nil {
		die("Failed to open %s: %v", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		die("Failed to create %s: %v", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		die("Failed to copy binary: %v", err)
	}
}
