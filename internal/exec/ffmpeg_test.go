package exec

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestClassifyError(t *testing.T) {
	cases := []struct {
		err  string
		want string
	}{
		{"401 Unauthorized", "auth"},
		{"Server returned 401 unauthorized", "auth"},
		{"Connection refused", "network-refused"},
		{"timed out", "network-timeout"},
		{"not found", "not-found"},
		{"weird message", "unknown"},
	}
	for _, c := range cases {
		if got := ClassifyError(c.err); got != c.want {
			t.Fatalf("ClassifyError(%q) got %s want %s", c.err, got, c.want)
		}
	}
}

func TestWithTimeoutZero(t *testing.T) {
	ctx, cancel := WithTimeout(context.Background(), 0)
	defer cancel()
	select {
	case <-ctx.Done():
		t.Fatalf("zero timeout context should not be canceled immediately")
	default:
	}
}

func TestHasBinary(t *testing.T) {
	if !HasBinary("go") {
		t.Fatalf("expected 'go' to be found in PATH")
	}
	if HasBinary("definitely_missing_binary_xyz") {
		t.Fatalf("expected missing binary to return false")
	}
}

func TestAdaptFFmpegArgs(t *testing.T) {
	t.Run("ffmpeg 9 vsync removed -> fps_mode passthrough", func(t *testing.T) {
		args := []string{"-i", "rtsp://cam", "-vsync", "0", "-frames:v", "1", "out.jpg"}
		stderr := "Unrecognized option 'vsync'.\nError splitting the argument list: Option not found\n"
		got, ok := adaptFFmpegArgs(args, stderr, nil)
		if !ok {
			t.Fatalf("expected adaptation to succeed")
		}
		want := []string{"-i", "rtsp://cam", "-fps_mode", "passthrough", "-frames:v", "1", "out.jpg"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("ffmpeg 4 fps_mode missing -> vsync 0", func(t *testing.T) {
		args := []string{"-i", "rtsp://cam", "-fps_mode", "passthrough", "-frames:v", "1", "out.jpg"}
		stderr := "Unrecognized option 'fps_mode'.\nError splitting the argument list: Option not found\n"
		got, ok := adaptFFmpegArgs(args, stderr, nil)
		if !ok {
			t.Fatalf("expected adaptation to succeed")
		}
		want := []string{"-i", "rtsp://cam", "-vsync", "0", "-frames:v", "1", "out.jpg"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("both fps_mode and vsync unsupported -> strips option", func(t *testing.T) {
		tried := map[string]bool{"fps_mode": true}
		args := []string{"-i", "rtsp://cam", "-vsync", "0", "-frames:v", "1", "out.jpg"}
		stderr := "Unrecognized option 'vsync'.\n"
		got, ok := adaptFFmpegArgs(args, stderr, tried)
		if !ok {
			t.Fatalf("expected adaptation to succeed")
		}
		want := []string{"-i", "rtsp://cam", "-frames:v", "1", "out.jpg"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("ffmpeg 4 rtsp timeout listen mode -> stimeout", func(t *testing.T) {
		args := []string{"-timeout", "7000000", "-i", "rtsp://cam", "out.jpg"}
		stderr := "[rtsp @ 0x123] Cannot use timeout without listen\n"
		got, ok := adaptFFmpegArgs(args, stderr, nil)
		if !ok {
			t.Fatalf("expected adaptation to succeed")
		}
		want := []string{"-stimeout", "7000000", "-i", "rtsp://cam", "out.jpg"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}

func TestRunFFmpegFallbackToLegacy(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "ffmpeg")
	// Simulate FFmpeg 4.x where -fps_mode is unrecognized, but -vsync 0 succeeds.
	content := `#!/bin/sh
for arg in "$@"; do
	if [ "$arg" = "-fps_mode" ]; then
		echo "Unrecognized option 'fps_mode'." >&2
		echo "Error splitting the argument list: Option not found" >&2
		exit 8
	fi
done
exit 0
`
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatalf("write stub ffmpeg: %v", err)
	}
	t.Setenv("PATH", dir)

	err := RunFFmpeg(context.Background(), "-i", "rtsp://127.0.0.1/cam", "-fps_mode", "passthrough", "out.jpg")
	if err != nil {
		t.Fatalf("expected RunFFmpeg to fall back from -fps_mode to -vsync and succeed, got: %v", err)
	}
}
