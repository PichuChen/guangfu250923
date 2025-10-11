package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Simple response wrapper
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// performUpdate downloads latest release asset and restarts systemd service.
func performUpdate(ctx context.Context, repo, assetPattern, service, installPath string, dryRun bool, openapiAsset, openapiDest string) (map[string]any, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github api status %d", resp.StatusCode)
	}
	var payload struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.TagName == "" {
		return nil, errors.New("missing tag_name in release")
	}
	var assetURL, assetName string
	var openapiURL string
	for _, a := range payload.Assets {
		if a.Name == "" || a.URL == "" {
			continue
		}
		// capture binary asset
		if assetURL == "" && (assetPattern == "" || strings.Contains(a.Name, assetPattern)) {
			assetURL = a.URL
			assetName = a.Name
		}
		// capture openapi asset (exact match by default)
		if openapiAsset != "" && a.Name == openapiAsset {
			openapiURL = a.URL
		}
	}
	if assetURL == "" && len(payload.Assets) > 0 { // fallback first if pattern not found
		assetURL = payload.Assets[0].URL
		assetName = payload.Assets[0].Name
	}
	if assetURL == "" {
		return nil, errors.New("no asset matched pattern")
	}
	// openapi asset is optional; only require if dest set AND asset name specified but not found
	if openapiDest != "" && openapiAsset != "" && openapiURL == "" {
		// warn but continue (do not fail) to avoid blocking binary security updates
		fmt.Fprintf(os.Stderr, "[updater] warning: openapi asset %s not found in release %s\n", openapiAsset, payload.TagName)
	}

	tmpDir, err := os.MkdirTemp("", "updater-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	newBin := filepath.Join(tmpDir, "new-bin")

	if dryRun {
		return map[string]any{"tag": payload.TagName, "asset": assetName, "openapi_asset": openapiAsset, "dry_run": true}, nil
	}

	ar, err := client.Get(assetURL)
	if err != nil {
		return nil, err
	}
	defer ar.Body.Close()
	if ar.StatusCode != 200 {
		return nil, fmt.Errorf("asset download status %d", ar.StatusCode)
	}
	f, err := os.Create(newBin)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(f, ar.Body); err != nil {
		f.Close()
		return nil, err
	}
	f.Close()
	if err := os.Chmod(newBin, 0o755); err != nil {
		return nil, err
	}

	// optional simple integrity marker (hash)
	hashFile, _ := os.Open(newBin)
	sha := sha256.New()
	io.Copy(sha, hashFile)
	hashFile.Close()
	sum := hex.EncodeToString(sha.Sum(nil))

	backupPath := installPath + ".bak." + time.Now().Format("20060102150405")
	if _, err := os.Stat(installPath); err == nil {
		_ = copyFile(installPath, backupPath)
	}
	if err := os.Rename(newBin, installPath); err != nil {
		return nil, err
	}

	if out, err := runCmd("systemctl", "stop", service); err != nil {
		return nil, fmt.Errorf("stop failed: %v / %s", err, out)
	}
	if out, err := runCmd("systemctl", "start", service); err != nil {
		return nil, fmt.Errorf("start failed: %v / %s", err, out)
	}
	// quick health check (optional) can be added here

	// Optionally download openapi spec asset
	var openapiWritten string
	if openapiURL != "" && openapiDest != "" {
		if err := os.MkdirAll(filepath.Dir(openapiDest), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "[updater] failed to create dir for openapi: %v\n", err)
		} else if err := downloadTo(client, openapiURL, openapiDest); err != nil {
			fmt.Fprintf(os.Stderr, "[updater] failed to download openapi asset: %v\n", err)
		} else {
			openapiWritten = openapiDest
		}
	}

	return map[string]any{
		"tag":           payload.TagName,
		"asset":         assetName,
		"backup":        backupPath,
		"sha256":        sum,
		"service":       service,
		"openapi_asset": openapiAsset,
		"openapi_path":  openapiWritten,
	}, nil
}

func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	b, err := cmd.CombinedOutput()
	return string(b), err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// downloadTo streams a URL to destination path.
func downloadTo(client *http.Client, url, dest string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	tmp := dest + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	f.Close()
	return os.Rename(tmp, dest)
}

func main() {
	addr := envOr("UPDATER_LISTEN", ":9090")
	apiKey := os.Getenv("UPDATE_API_KEY")
	if apiKey == "" {
		fmt.Println("[WARN] UPDATE_API_KEY not set; all requests will be rejected")
	}
	service := envOr("SERVICE_NAME", "guangfu250923")
	install := envOr("INSTALL_PATH", "/etc/guangfu250923/guangfu250923")
	repo := envOr("GITHUB_REPO", "PichuChen/guangfu250923")
	pattern := os.Getenv("ASSET_PATTERN") // optional
	openapiAsset := envOr("OPENAPI_ASSET_NAME", "openapi.yaml")
	openapiDest := envOr("OPENAPI_DEST_PATH", "/etc/guangfu250923/openapi.yaml")
	if openapiDest == "" {
		// default: same directory as binary if install has directory portion
		openapiDest = filepath.Join(filepath.Dir(install), "openapi.yaml")
	}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("ok")) })

	http.HandleFunc("/upgrade-service", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("upgrade request", "from", r.RemoteAddr, "user-agent", r.UserAgent())
		key := r.Header.Get("X-API-Key")
		if key == "" {
			key = r.URL.Query().Get("key")
		}
		if apiKey == "" || key != apiKey {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		dryRun := r.URL.Query().Get("dry_run") == "true"
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		res, err := performUpdate(ctx, repo, pattern, service, install, dryRun, openapiAsset, openapiDest)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, res)
	})

	fmt.Printf("[updater] listening on %s (service=%s repo=%s)\n", addr, service, repo)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintf(os.Stderr, "listen error: %v\n", err)
		os.Exit(1)
	}
}
