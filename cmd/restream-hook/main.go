package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type relayConfig struct {
	RoomID       string `json:"roomId"`
	SourceURL    string `json:"sourceUrl"`
	Destinations []struct {
		Label string `json:"label"`
		URL   string `json:"url"`
	} `json:"destinations"`
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("usage: restream-hook publish|unpublish <path>")
	}
	action := strings.TrimSpace(os.Args[1])
	path := strings.TrimSpace(os.Args[2])
	room := strings.TrimPrefix(path, "live/")

	switch action {
	case "publish":
		if err := startRelay(room); err != nil {
			log.Printf("publish %s: %v", path, err)
			os.Exit(1)
		}
	case "unpublish":
		stopRelay(room)
	default:
		log.Fatalf("unknown action: %s", action)
	}
}

func startRelay(room string) error {
	stopRelay(room)

	cfg, err := fetchRelay(room)
	if err != nil {
		return err
	}
	if len(cfg.Destinations) == 0 {
		return fmt.Errorf("no destinations configured")
	}

	args := []string{
		"-hide_banner",
		"-loglevel", "warning",
		"-i", cfg.SourceURL,
	}
	for _, dest := range cfg.Destinations {
		args = append(args, "-c", "copy", "-f", "flv", dest.URL)
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}

	pidPath := pidFile(room)
	if err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0o644); err != nil {
		_ = cmd.Process.Kill()
		return err
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("ffmpeg room %s exited: %v", room, err)
		}
		_ = os.Remove(pidPath)
	}()

	log.Printf("relay started room=%s pid=%d destinations=%d", room, cmd.Process.Pid, len(cfg.Destinations))
	time.Sleep(300 * time.Millisecond)
	return nil
}

func stopRelay(room string) {
	pidPath := pidFile(room)
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return
	}
	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil || pid <= 0 {
		_ = os.Remove(pidPath)
		return
	}
	proc, err := os.FindProcess(pid)
	if err == nil {
		_ = proc.Signal(syscall.SIGTERM)
	}
	_ = os.Remove(pidPath)
}

func fetchRelay(room string) (relayConfig, error) {
	api := strings.TrimRight(strings.TrimSpace(os.Getenv("RESTREAM_API_URL")), "/")
	if api == "" {
		api = "http://127.0.0.1:8080"
	}
	token := strings.TrimSpace(os.Getenv("RESTREAM_INTERNAL_TOKEN"))
	if token == "" {
		return relayConfig{}, fmt.Errorf("RESTREAM_INTERNAL_TOKEN not set")
	}

	req, err := http.NewRequest(http.MethodGet, api+"/internal/restream/relay/"+room, nil)
	if err != nil {
		return relayConfig{}, err
	}
	req.Header.Set("X-Restream-Internal-Token", token)

	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return relayConfig{}, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return relayConfig{}, fmt.Errorf("relay config %s: %s", res.Status, strings.TrimSpace(string(body)))
	}

	var cfg relayConfig
	if err := json.Unmarshal(body, &cfg); err != nil {
		return relayConfig{}, err
	}
	return cfg, nil
}

func pidFile(room string) string {
	return filepath.Join(os.TempDir(), "woragis-restream-"+room+".pid")
}
