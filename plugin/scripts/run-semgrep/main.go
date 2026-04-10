// Semgrep fragment scanner hook for Claude Code.
//
// Reads a file_path from stdin (Claude post-tool hook JSON), loads the file(s),
// posts a scan request to the semgrep fragment endpoint, and prints a
// PostToolHookResponse JSON to stdout.
//
// Usage: go run . scan [--config <rule-file>] [--<flag>] [<extra-files>...]
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PostToolHookResponse mirrors the Claude hook response schema.
type PostToolHookResponse struct {
	Decision *string `json:"decision,omitempty"`
	Reason   *string `json:"reason,omitempty"`
}

func blockResponse(reason string) PostToolHookResponse {
	d := "block"
	return PostToolHookResponse{Decision: &d, Reason: &reason}
}

func allowResponse(reason string) PostToolHookResponse {
	return PostToolHookResponse{Reason: &reason}
}

// loadFilePathFromStdin reads the Claude hook JSON from stdin and returns tool_input.file_path.
func loadFilePathFromStdin() (string, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	var hook struct {
		ToolInput struct {
			FilePath string `json:"file_path"`
		} `json:"tool_input"`
	}
	if err := json.Unmarshal(data, &hook); err != nil {
		return "", err
	}
	return hook.ToolInput.FilePath, nil
}

func userDataFolder() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome != "" {
		if info, err := os.Stat(configHome); err == nil && info.IsDir() {
			return filepath.Join(configHome, ".semgrep")
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".semgrep")
}

func userSettingsFile() string {
	if path := os.Getenv("SEMGREP_SETTINGS_FILE"); path != "" {
		return path
	}
	return filepath.Join(userDataFolder(), "settings.yml")
}

// getAppTokenFromSettings reads api_token from the semgrep settings YAML file.
func getAppTokenFromSettings() string {
	data, err := os.ReadFile(userSettingsFile())
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "api_token:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "api_token:"))
			return strings.Trim(val, `'"`)
		}
	}
	return ""
}

// loadFiles reads the contents of the given paths (expanding directories).
// Returns a map of relative-path → file-contents.
func loadFiles(base string, names []string) map[string]string {
	files := map[string]string{}

	for _, name := range names {
		p, err := filepath.Abs(name)
		if err != nil {
			continue
		}
		info, err := os.Stat(p)
		if err != nil {
			continue
		}

		if info.IsDir() {
			filepath.Walk(p, func(fp string, fi os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				// Skip hidden directories and files
				for _, part := range strings.Split(fp, string(os.PathSeparator)) {
					if strings.HasPrefix(part, ".") && part != "." {
						if fi.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
				}
				if fi.IsDir() {
					return nil
				}
				rel, err := filepath.Rel(base, fp)
				if err != nil {
					return nil
				}
				data, err := os.ReadFile(fp)
				if err != nil {
					return nil // skip binary / unreadable files
				}
				if !isValidUTF8(data) {
					return nil
				}
				files[rel] = string(data)
				return nil
			})
		} else {
			rel, err := filepath.Rel(base, p)
			if err != nil {
				continue
			}
			data, err := os.ReadFile(p)
			if err != nil || !isValidUTF8(data) {
				continue
			}
			files[rel] = string(data)
		}
	}
	return files
}

func isValidUTF8(b []byte) bool {
	// Fast check: valid UTF-8 has no null bytes and no invalid sequences.
	// os.ReadFile text check: attempt to convert cleanly.
	for _, c := range b {
		if c == 0 {
			return false
		}
	}
	return true
}

// requestScan posts the scan payload to url, retrying on connection/service errors.
func requestScan(url string, payload interface{}, appToken string) (map[string]interface{}, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5*time.Minute + 5*time.Second}

	for {
		req, err := http.NewRequest("POST", url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if appToken != "" {
			req.Header.Set("Authorization", "Bearer "+appToken)
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "connection error:", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			return nil, fmt.Errorf("unauthorized (401)")
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var result map[string]interface{}
			err := json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()
			return result, err
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Fprintln(os.Stderr, "service error:", string(respBody))
		time.Sleep(500 * time.Millisecond)
	}
}

func printJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.Encode(v)
}

func main() {
	localURL := "http://127.0.0.1:8000/api/run"
	url := os.Getenv("SEMGREP_FRAGMENT_URL")
	if url == "" {
		url = localURL
	}

	args := os.Args[1:]
	if len(args) == 0 || args[0] != "scan" {
		fmt.Fprintf(os.Stderr, "error: use %s scan ...\n", os.Args[0])
		os.Exit(-1)
	}
	args = args[1:]

	config := map[string]interface{}{}
	var extraFiles []string

	for len(args) > 0 {
		arg := args[0]
		args = args[1:]

		if strings.HasPrefix(arg, "--") {
			key := strings.TrimPrefix(arg, "--")
			if key == "config" && len(args) > 0 {
				rulePath := args[0]
				args = args[1:]
				ruleData, err := os.ReadFile(rulePath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error reading config file: %v\n", err)
					os.Exit(1)
				}
				config["rule"] = string(ruleData)
			} else {
				config[key] = true
			}
		} else {
			extraFiles = append(extraFiles, arg)
		}
	}

	// Get file_path from Claude hook stdin
	filePath, err := loadFilePathFromStdin()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading stdin:", err)
		os.Exit(1)
	}
	extraFiles = append(extraFiles, filePath)

	// Resolve app token
	appToken := os.Getenv("SEMGREP_APP_TOKEN")
	if appToken == "" {
		appToken = getAppTokenFromSettings()
	}

	if appToken == "" {
		reason := "No app token found. You might have to restart your Claude session and activate your Semgrep session in your browser. You should not have to run `semgrep login` manually, a browser window will open at the beginning of the Claude session."
		printJSON(blockResponse(reason))
		os.Exit(0) // exit 0 to show JSON response to user
	}

	config["app_token"] = appToken

	cwd, _ := os.Getwd()
	scanFiles := loadFiles(cwd, extraFiles)

	payload := map[string]interface{}{
		"command": map[string]interface{}{
			"name":   "scan",
			"files":  scanFiles,
			"config": config,
			"trace":  nil,
		},
	}

	response, err := requestScan(url, payload, appToken)
	if err != nil {
		reason := fmt.Sprintf("Scan request failed: %v", err)
		printJSON(blockResponse(reason))
		os.Exit(0)
	}

	result, _ := response["result"].(map[string]interface{})
	if result == nil {
		printJSON(allowResponse("No results"))
		return
	}

	resultJSON, _ := result["json"].(map[string]interface{})
	if resultJSON == nil {
		printJSON(allowResponse("No results"))
		return
	}

	findingsRaw, _ := resultJSON["results"].([]interface{})
	if len(findingsRaw) == 0 {
		printJSON(allowResponse("No findings"))
		return
	}

	type finding struct {
		Line        interface{} `json:"line"`
		DisplayName interface{} `json:"display_name"`
		Message     interface{} `json:"message"`
		Severity    interface{} `json:"severity"`
		CWE         interface{} `json:"cwe"`
	}

	var findings []finding
	for _, raw := range findingsRaw {
		r, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		start, _ := r["start"].(map[string]interface{})
		extra, _ := r["extra"].(map[string]interface{})
		metadata, _ := extra["metadata"].(map[string]interface{})

		var line, displayName, message, severity, cwe interface{}
		if start != nil {
			line = start["line"]
		}
		if metadata != nil {
			displayName = metadata["display-name"]
			cwe = metadata["cwe"]
		}
		if extra != nil {
			message = extra["message"]
			severity = extra["severity"]
		}
		findings = append(findings, finding{
			Line:        line,
			DisplayName: displayName,
			Message:     message,
			Severity:    severity,
			CWE:         cwe,
		})
	}

	reasonBytes, _ := json.Marshal(findings)
	reason := string(reasonBytes)
	printJSON(blockResponse(reason))
}
