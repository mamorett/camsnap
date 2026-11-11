// Package exec wraps external command execution helpers.
package exec

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RunFFmpeg executes ffmpeg with a timeout, returning combined output on error.
func RunFFmpeg(ctx context.Context, args ...string) error {
	_, err := RunFFmpegWithOutput(ctx, args...)
	return err
}

// RunFFmpegWithOutput returns combined stdout/stderr output alongside error.
// If ffmpeg fails due to version-specific CLI options (e.g. -fps_mode vs -vsync,
// or -timeout vs -stimeout across FFmpeg releases), it automatically adapts the
// arguments and retries.
func RunFFmpegWithOutput(ctx context.Context, args ...string) (string, error) {
	curArgs := append([]string(nil), args...)
	tried := make(map[string]bool)
	const maxAttempts = 6

	for attempt := 0; attempt < maxAttempts; attempt++ {
		cmd := exec.CommandContext(ctx, "ffmpeg", curArgs...)
		output, err := cmd.CombinedOutput()
		outStr := string(output)
		if err == nil {
			return outStr, nil
		}
		if ctx.Err() != nil {
			return outStr, fmt.Errorf("ffmpeg %v failed: %w\n%s", curArgs, err, outStr)
		}
		nextArgs, ok := adaptFFmpegArgs(curArgs, outStr, tried)
		if !ok {
			return outStr, fmt.Errorf("ffmpeg %v failed: %w\n%s", curArgs, err, outStr)
		}
		curArgs = nextArgs
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", curArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("ffmpeg %v failed: %w\n%s", curArgs, err, string(output))
	}
	return string(output), nil
}

// adaptFFmpegArgs rewrites or removes version-specific ffmpeg flags when stderr
// indicates an option is unrecognized or incompatible with the installed ffmpeg version.
func adaptFFmpegArgs(args []string, stderr string, tried map[string]bool) ([]string, bool) {
	if tried == nil {
		tried = make(map[string]bool)
	}
	lower := strings.ToLower(stderr)

	// In FFmpeg < 5.0, RTSP -timeout was a server listen-mode option.
	if strings.Contains(lower, "without listen") || strings.Contains(lower, "open rtsp for listening") {
		tried["timeout"] = true
		if tried["stimeout"] {
			return removeOptionPair(args, "-timeout")
		}
		if next, ok := replaceOptionPair(args, "-timeout", "-stimeout", ""); ok {
			return next, true
		}
	}

	opt := extractUnrecognizedOption(stderr)
	if opt == "" {
		return nil, false
	}
	tried[opt] = true

	switch opt {
	case "fps_mode":
		if tried["vsync"] {
			return removeOptionPair(args, "-fps_mode")
		}
		return replaceOptionPair(args, "-fps_mode", "-vsync", "0")
	case "vsync":
		if tried["fps_mode"] {
			return removeOptionPair(args, "-vsync")
		}
		return replaceOptionPair(args, "-vsync", "-fps_mode", "passthrough")
	case "timeout":
		if tried["stimeout"] {
			return removeOptionPair(args, "-timeout")
		}
		return replaceOptionPair(args, "-timeout", "-stimeout", "")
	case "stimeout":
		return removeOptionPair(args, "-stimeout")
	case "skip_frame", "flags2", "analyzeduration", "probesize", "update":
		return removeOptionPair(args, "-"+opt)
	default:
		return nil, false
	}
}

func extractUnrecognizedOption(stderr string) string {
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)

		// Matches: Unrecognized option 'vsync'.
		if idx := strings.Index(lower, "unrecognized option '"); idx >= 0 {
			rest := line[idx+len("unrecognized option '"):]
			if end := strings.IndexByte(rest, '\''); end > 0 {
				return strings.TrimPrefix(rest[:end], "-")
			}
		}

		// Matches: Option vsync not found. / Option 'vsync' not found.
		if strings.HasPrefix(lower, "option ") && strings.Contains(lower, " not found") {
			rest := line[len("option "):]
			if end := strings.Index(strings.ToLower(rest), " not found"); end > 0 {
				opt := strings.TrimSpace(rest[:end])
				opt = strings.Trim(opt, "'\"")
				return strings.TrimPrefix(opt, "-")
			}
		}

		// Matches: Failed to set value '...' for option 'vsync': Option not found
		if idx := strings.Index(lower, "for option '"); idx >= 0 && strings.Contains(lower, "not found") {
			rest := line[idx+len("for option '"):]
			if end := strings.IndexByte(rest, '\''); end > 0 {
				return strings.TrimPrefix(rest[:end], "-")
			}
		}
	}
	return ""
}

func replaceOptionPair(args []string, oldFlag, newFlag, newVal string) ([]string, bool) {
	for i := 0; i < len(args); i++ {
		if args[i] == oldFlag {
			next := make([]string, 0, len(args))
			next = append(next, args[:i]...)
			next = append(next, newFlag)
			if i+1 < len(args) {
				if newVal != "" {
					next = append(next, newVal)
				} else {
					next = append(next, args[i+1])
				}
				next = append(next, args[i+2:]...)
			}
			return next, true
		}
	}
	return nil, false
}

func removeOptionPair(args []string, flag string) ([]string, bool) {
	for i := 0; i < len(args); i++ {
		if args[i] == flag {
			next := make([]string, 0, len(args))
			next = append(next, args[:i]...)
			if i+1 < len(args) {
				next = append(next, args[i+2:]...)
			}
			return next, true
		}
	}
	return nil, false
}

// WithTimeout returns a derived context that times out.
func WithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, d)
}

// HasBinary reports whether a named executable is available in PATH.
func HasBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// ClassifyError inspects ffmpeg stderr to differentiate auth vs network errors.
func ClassifyError(stderr string) string {
	lower := strings.ToLower(stderr)
	switch {
	case strings.Contains(lower, "401") || strings.Contains(lower, "unauthorized") || strings.Contains(lower, "auth"):
		return "auth"
	case strings.Contains(lower, "connection refused"):
		return "network-refused"
	case strings.Contains(lower, "timed out") || strings.Contains(lower, "timeout"):
		return "network-timeout"
	case strings.Contains(lower, "not found"):
		return "not-found"
	default:
		return "unknown"
	}
}
