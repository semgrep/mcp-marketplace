// Standalone semgrep login program.
//
// Opens a browser for the user to authenticate with semgrep.dev, polls for the
// resulting token, validates it, and writes it to ~/.semgrep/settings.yml.
//
// Usage: go run . (or compile with go build)
package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	waitBetweenRetrySec = 6
	maxRetries          = 30 // ~3 minutes
)

func semgrepURL() string {
	if u := os.Getenv("SEMGREP_URL"); u != "" {
		return u
	}
	return "https://semgrep.dev"
}

func getSettingsPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "semgrep", "settings.yml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".semgrep", "settings.yml")
}

// readToken extracts the api_token value from a simple YAML settings file.
// Returns "" if not found or file doesn't exist.
func readToken(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "api_token:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "api_token:"))
			// Strip optional surrounding quotes
			val = strings.Trim(val, `'"`)
			return val
		}
	}
	return ""
}

// writeToken writes (or updates) api_token in the settings YAML file,
// preserving any other existing keys.
func writeToken(path, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	var lines []string
	if data, err := os.ReadFile(path); err == nil {
		lines = strings.Split(string(data), "\n")
		// Remove trailing empty element from split
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
	}

	found := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "api_token:") {
			lines[i] = "api_token: " + token
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, "api_token: "+token)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "settings*.yml")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	_, err = fmt.Fprintln(tmp, strings.Join(lines, "\n"))
	tmp.Close()
	if err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

func validateToken(token string) bool {
	if token == "" {
		return false
	}
	req, err := http.NewRequest("GET", semgrepURL()+"/api/agent/deployments/current", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant RFC4122
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "cmd", []string{"/c", "start", url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}

var hexToken = regexp.MustCompile(`^[0-9a-f]+$`)

func main() {
	settingsPath := getSettingsPath()

	existing := readToken(settingsPath)
	if existing != "" && validateToken(existing) {
		fmt.Printf("Already logged in. Token saved at %s.\n", settingsPath)
		fmt.Println("Run `semgrep logout` first if you want to log in again.")
		os.Exit(0)
	}

	sessionID := generateUUID()
	loginURL := fmt.Sprintf("%s/login?cli-token=%s", semgrepURL(), sessionID)

	fmt.Println("Opening browser to log in to semgrep.dev...")
	fmt.Printf("  %s\n", loginURL)
	openBrowser(loginURL)
	fmt.Println("\nWaiting for login... (you have ~3 minutes)\n")

	client := &http.Client{Timeout: 10 * time.Second}
	pollURL := semgrepURL() + "/api/agent/tokens/requests"

	for attempt := 0; attempt < maxRetries; attempt++ {
		body, _ := json.Marshal(map[string]string{"token_request_key": sessionID})
		resp, err := client.Post(pollURL, "application/json", bytes.NewReader(body))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Semgrep login: Network error: %v\n", err)
			os.Exit(2)
		}

		switch resp.StatusCode {
		case http.StatusOK:
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var result map[string]interface{}
			if err := json.Unmarshal(respBody, &result); err != nil {
				fmt.Fprintln(os.Stderr, "Semgrep login: Error: failed to parse server response.")
				os.Exit(2)
			}

			token, _ := result["token"].(string)
			if token == "" {
				fmt.Fprintln(os.Stderr, "Semgrep login: Error: server returned 200 but no token in response.")
				os.Exit(2)
			}
			if len(token) != 64 || !hexToken.MatchString(token) {
				fmt.Fprintln(os.Stderr, "Semgrep login: Error: received token has unexpected format.")
				os.Exit(2)
			}

			fmt.Println("Token received. Validating...")
			if !validateToken(token) {
				fmt.Fprintln(os.Stderr, "Semgrep login: Error: token validation failed.")
				os.Exit(2)
			}

			if err := writeToken(settingsPath, token); err != nil {
				fmt.Fprintf(os.Stderr, "Semgrep login: Error writing token: %v\n", err)
				os.Exit(2)
			}
			fmt.Printf("Logged in. Token saved to %s.\n", settingsPath)
			os.Exit(0)

		case http.StatusNotFound:
			resp.Body.Close()
			// User hasn't completed browser login yet — keep polling.

		default:
			resp.Body.Close()
			fmt.Fprintf(os.Stderr, "Semgrep login: Unexpected response from server: %d\n", resp.StatusCode)
			os.Exit(2)
		}

		fmt.Printf("  Waiting... (%d/%d)\r", attempt+1, maxRetries)
		time.Sleep(waitBetweenRetrySec * time.Second)
	}

	fmt.Fprintln(os.Stderr, "\nSemgrep login: Login timed out. Please try again.")
	os.Exit(2)
}
