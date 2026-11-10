package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/steipete/camsnap/internal/config"
)

func TestAddAndList(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "camsnap", "config.yaml")

	root := NewRootCommand("test")
	root.SetArgs([]string{"--config", cfgPath, "add", "--name", "t1", "--host", "1.1.1.1", "--user", "u", "--pass", "p"})
	if err := root.Execute(); err != nil {
		t.Fatalf("add execute: %v", err)
	}

	var buf bytes.Buffer
	root = NewRootCommand("test")
	root.SetOut(&buf)
	root.SetArgs([]string{"--config", cfgPath, "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("list execute: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("t1")) {
		t.Fatalf("expected camera name in list output, got: %s", buf.String())
	}
}

func TestSnapNoFFmpeg(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "camsnap", "config.yaml")
	// write config with one camera
	cfg := config.Config{
		Cameras: []config.Camera{{
			Name:          "cam",
			Host:          "127.0.0.1",
			Port:          554,
			Protocol:      "rtsp",
			Username:      "u",
			Password:      "p",
			RTSPTransport: "udp",
			Stream:        "stream1",
		}},
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	// empty PATH to ensure ffmpeg not found
	t.Setenv("PATH", "")
	root := NewRootCommand("test")
	root.SetArgs([]string{"--config", cfgPath, "snap", "cam", "--out", filepath.Join(t.TempDir(), "snap.jpg")})
	if err := root.Execute(); err == nil {
		t.Fatalf("expected error due to missing ffmpeg")
	}
}

func TestSnapCreatesTempFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "camsnap", "config.yaml")
	cfg := config.Config{
		Cameras: []config.Camera{{
			Name:     "cam",
			Host:     "127.0.0.1",
			Port:     554,
			Protocol: "rtsp",
			Username: "u",
			Password: "p",
		}},
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	ffmpegPath := makeStubFFmpeg(t)
	t.Setenv("PATH", ffmpegPath)

	root := NewRootCommand("test")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"--config", cfgPath, "snap", "cam"})
	if err := root.Execute(); err != nil {
		t.Fatalf("snap: %v", err)
	}
	out := buf.String()
	path := extractTempPath(t, out)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected temp snap file to exist: %v", err)
	}
}

func TestRootVersionHelp(t *testing.T) {
	root := NewRootCommand("test-version")
	root.SetArgs([]string{"--version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("--version: %v", err)
	}

	root = NewRootCommand("test-version")
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("--help: %v", err)
	}
}

func TestDoctorNoCameras(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCommand("test")
	root.SetArgs([]string{"doctor"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doctor: %v", err)
	}
}

func TestDiscoverNoDevices(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCommand("test")
	root.SetArgs([]string{"discover", "--timeout", "10ms"})
	if err := root.Execute(); err != nil {
		t.Fatalf("discover: %v", err)
	}
}

func TestWatchMissingAction(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "camsnap", "config.yaml")
	cfg := config.Config{
		Cameras: []config.Camera{{
			Name:     "cam",
			Host:     "127.0.0.1",
			Username: "u",
			Password: "p",
		}},
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	root := NewRootCommand("test")
	root.SetArgs([]string{"--config", cfgPath, "watch", "cam"})
	if err := root.Execute(); err == nil {
		t.Fatalf("expected error for missing action")
	}
}

func TestClipTempOutput(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "camsnap", "config.yaml")
	cfg := config.Config{
		Cameras: []config.Camera{{
			Name:          "cam",
			Host:          "127.0.0.1",
			Port:          554,
			Protocol:      "rtsp",
			Username:      "u",
			Password:      "p",
			RTSPTransport: "tcp",
		}},
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	ffmpegPath := makeStubFFmpeg(t)
	t.Setenv("PATH", ffmpegPath)

	root := NewRootCommand("test")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"--config", cfgPath, "clip", "cam", "--dur", "1s", "--timeout", "2s"})

	if err := root.Execute(); err != nil {
		t.Fatalf("clip: %v", err)
	}
	path := extractTempPath(t, buf.String())
	if info, err := os.Stat(path); err != nil {
		t.Fatalf("expected temp clip file to exist: %v", err)
	} else if info.Size() != 0 {
		// stub ffmpeg writes empty file; size zero is expected. Any size is fine but should exist.
		_ = info
	}
}

func makeStubFFmpeg(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "ffmpeg")
	content := []byte("#!/bin/sh\nout=\"\"\nfor last in \"$@\"; do\n\tout=\"$last\"\ndone\n: >\"$out\"\nexit 0\n")
	if err := os.WriteFile(script, content, 0o755); err != nil {
		t.Fatalf("write stub ffmpeg: %v", err)
	}
	return dir
}

func extractTempPath(t *testing.T, output string) string {
	t.Helper()
	lines := bytes.Split([]byte(output), []byte("\n"))
	for _, l := range lines {
		if bytes.Contains(l, []byte("writing")) {
			parts := bytes.Fields(l)
			if len(parts) > 0 {
				// path is last token
				return string(parts[len(parts)-1])
			}
		}
	}
	t.Fatalf("could not extract temp path from output: %s", output)
	return ""
}

func makeLoggingStubFFmpeg(t *testing.T, logFile string) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "ffmpeg")
	content := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$*\" >> %q\nout=\"\"\nfor last in \"$@\"; do\n\tout=\"$last\"\ndone\n: >\"$out\"\nexit 0\n", logFile)
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatalf("write stub ffmpeg: %v", err)
	}
	return dir
}

func TestAudioCommandDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "camsnap", "config.yaml")
	cfg := config.Config{
		Cameras: []config.Camera{{
			Name:          "robot",
			Host:          "192.168.0.48",
			Port:          8554,
			Protocol:      "rtsp",
			Path:          "/robot",
			RTSPTransport: "tcp",
		}},
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	logFile := filepath.Join(t.TempDir(), "ffmpeg.log")
	t.Setenv("PATH", makeLoggingStubFFmpeg(t, logFile))

	root := NewRootCommand("test")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"--config", cfgPath, "audio", "robot"})

	if err := root.Execute(); err != nil {
		t.Fatalf("audio: %v", err)
	}

	out := buf.String()
	path := extractTempPath(t, out)
	if !bytes.HasSuffix([]byte(path), []byte(".wav")) {
		t.Fatalf("expected .wav temp file extension, got %s", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected audio file to exist: %v", err)
	}

	logContent, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	args := string(logContent)

	// Verify ffmpeg arguments: -n (do not overwrite), -vn (no video), -c:a copy, -t 10 (default 10s)
	for _, expected := range []string{"-n", "-rtsp_transport tcp", "-i rtsp://192.168.0.48:8554/robot", "-t 10", "-vn", "-c:a copy"} {
		if !bytes.Contains([]byte(args), []byte(expected)) {
			t.Errorf("expected arg %q in ffmpeg invocation: %s", expected, args)
		}
	}
}

func TestAudioCommandSeconds(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "camsnap", "config.yaml")
	cfg := config.Config{
		Cameras: []config.Camera{{
			Name:          "robot",
			Host:          "192.168.0.48",
			Port:          8554,
			Protocol:      "rtsp",
			Path:          "/robot",
			RTSPTransport: "tcp",
		}},
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	logFile := filepath.Join(t.TempDir(), "ffmpeg.log")
	t.Setenv("PATH", makeLoggingStubFFmpeg(t, logFile))

	targetWav := filepath.Join(t.TempDir(), "custom_audio.wav")
	root := NewRootCommand("test")
	root.SetArgs([]string{"--config", cfgPath, "audio", "robot", "--seconds", "5", "--out", targetWav})

	if err := root.Execute(); err != nil {
		t.Fatalf("audio: %v", err)
	}

	logContent, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	args := string(logContent)

	for _, expected := range []string{"-t 5", "-vn", "-c:a copy", targetWav} {
		if !bytes.Contains([]byte(args), []byte(expected)) {
			t.Errorf("expected arg %q in ffmpeg invocation: %s", expected, args)
		}
	}
}

func TestAudioDoNotOverwriteExisting(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "camsnap", "config.yaml")
	cfg := config.Config{
		Cameras: []config.Camera{{
			Name:     "cam",
			Host:     "127.0.0.1",
			Protocol: "rtsp",
		}},
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	existingWav := filepath.Join(t.TempDir(), "existing.wav")
	if err := os.WriteFile(existingWav, []byte("initial-content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	t.Setenv("PATH", makeStubFFmpeg(t))

	root := NewRootCommand("test")
	root.SetArgs([]string{"--config", cfgPath, "audio", "cam", "--out", existingWav})
	if err := root.Execute(); err == nil {
		t.Fatalf("expected error when output file already exists without --overwrite")
	}

	// Content must not have been modified
	data, _ := os.ReadFile(existingWav)
	if string(data) != "initial-content" {
		t.Fatalf("file content changed despite do not overwrite rule")
	}

	// With --overwrite, it should succeed
	root = NewRootCommand("test")
	root.SetArgs([]string{"--config", cfgPath, "audio", "cam", "--out", existingWav, "--overwrite"})
	if err := root.Execute(); err != nil {
		t.Fatalf("expected overwrite to succeed, got %v", err)
	}
}

func TestClipAudioOnly(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "camsnap", "config.yaml")
	cfg := config.Config{
		Cameras: []config.Camera{{
			Name:     "cam",
			Host:     "127.0.0.1",
			Protocol: "rtsp",
		}},
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	logFile := filepath.Join(t.TempDir(), "ffmpeg.log")
	t.Setenv("PATH", makeLoggingStubFFmpeg(t, logFile))

	root := NewRootCommand("test")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"--config", cfgPath, "clip", "cam", "--audio-only", "--dur", "7s"})

	if err := root.Execute(); err != nil {
		t.Fatalf("clip audio-only: %v", err)
	}

	path := extractTempPath(t, buf.String())
	if !bytes.HasSuffix([]byte(path), []byte(".wav")) {
		t.Fatalf("expected .wav temp file extension, got %s", path)
	}

	logContent, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	args := string(logContent)

	for _, expected := range []string{"-vn", "-c:a copy", "-t 7"} {
		if !bytes.Contains([]byte(args), []byte(expected)) {
			t.Errorf("expected arg %q in ffmpeg invocation: %s", expected, args)
		}
	}
}

func TestAudioFlagsConflict(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "camsnap", "config.yaml")
	cfg := config.Config{
		Cameras: []config.Camera{{
			Name:     "cam",
			Host:     "127.0.0.1",
			Protocol: "rtsp",
		}},
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	t.Setenv("PATH", makeStubFFmpeg(t))

	// Conflicting --dur and --seconds
	root := NewRootCommand("test")
	root.SetArgs([]string{"--config", cfgPath, "audio", "cam", "--dur", "5s", "--seconds", "10"})
	if err := root.Execute(); err == nil {
		t.Fatalf("expected error for conflicting --dur and --seconds")
	}

	// Conflicting --audio-only and --no-audio
	root = NewRootCommand("test")
	root.SetArgs([]string{"--config", cfgPath, "clip", "cam", "--audio-only", "--no-audio"})
	if err := root.Execute(); err == nil {
		t.Fatalf("expected error for conflicting --audio-only and --no-audio")
	}
}
