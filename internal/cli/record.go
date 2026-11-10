package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/steipete/camsnap/internal/exec"
	"github.com/steipete/camsnap/internal/rtsp"
)

type recordOptions struct {
	cameraName   string
	outPath      string
	duration     time.Duration
	seconds      int
	timeout      time.Duration
	authMode     string
	transport    string
	stream       string
	path         string
	noAudio      bool
	audioOnly    bool
	audioCodec   string
	overwrite    bool
	slackChannel string
	slackToken   string
	slackMessage string
}

func executeRecord(cmd *cobra.Command, args []string, opts recordOptions) error {
	if opts.cameraName == "" && len(args) > 0 {
		opts.cameraName = args[0]
	}
	if opts.cameraName == "" {
		return fmt.Errorf("--camera is required")
	}

	if cmd.Flags().Changed("seconds") && cmd.Flags().Changed("dur") {
		return fmt.Errorf("cannot specify both --dur and --seconds")
	}
	if cmd.Flags().Changed("seconds") {
		if opts.seconds <= 0 {
			return fmt.Errorf("--seconds must be > 0")
		}
		opts.duration = time.Duration(opts.seconds) * time.Second
	}
	if opts.duration <= 0 {
		return fmt.Errorf("--dur must be > 0")
	}

	if !exec.HasBinary("ffmpeg") {
		return fmt.Errorf("ffmpeg not found in PATH")
	}

	if opts.audioOnly && opts.noAudio {
		return fmt.Errorf("cannot combine --audio-only and --no-audio")
	}

	if opts.outPath == "" {
		ext := "mp4"
		label := "clip"
		if opts.audioOnly {
			ext = "wav"
			label = "audio"
		}
		tmp, err := os.CreateTemp("", fmt.Sprintf("camsnap-*.%s", ext))
		if err != nil {
			return fmt.Errorf("create temp file: %w", err)
		}
		if err := tmp.Close(); err != nil {
			return fmt.Errorf("close temp file: %w", err)
		}
		opts.outPath = tmp.Name()
		if !opts.overwrite {
			// Remove the empty placeholder file so ffmpeg -n does not treat it as a collision.
			_ = os.Remove(opts.outPath)
		}
		cmd.Printf("No --out provided, writing %s to %s\n", label, opts.outPath)
	} else if !opts.overwrite {
		if _, err := os.Stat(opts.outPath); err == nil {
			return fmt.Errorf("output file %q already exists (use --overwrite to replace)", opts.outPath)
		}
	}

	cfgFlag, err := configPathFlag(cmd)
	if err != nil {
		return err
	}
	cfg, _, err := loadConfig(cfgFlag)
	if err != nil {
		return err
	}
	cam, ok := findCamera(cfg, opts.cameraName)
	if !ok {
		return fmt.Errorf("camera %q not found", opts.cameraName)
	}

	if opts.stream != "" && opts.path != "" {
		return fmt.Errorf("use --path for custom RTSP token URLs; omit --stream")
	}

	if opts.path == "" && cam.Path != "" {
		opts.path = cam.Path
	}
	if opts.path != "" {
		cam.Path = opts.path
		cam.Stream = ""
	}

	// Per-camera defaults
	if opts.transport == "" && cam.RTSPTransport != "" {
		opts.transport = cam.RTSPTransport
	}
	if opts.stream == "" && cam.Stream != "" && opts.path == "" {
		opts.stream = cam.Stream
	}

	if opts.audioOnly {
		// When recording audio only, disregard camera's NoAudio setting
		opts.noAudio = false
		if opts.audioCodec == "" && cam.AudioCodec != "" {
			opts.audioCodec = cam.AudioCodec
		}
		if opts.audioCodec == "" {
			opts.audioCodec = "copy"
		}
	} else {
		if !opts.noAudio && cam.NoAudio {
			opts.noAudio = true
		}
		if opts.audioCodec == "" && cam.AudioCodec != "" {
			opts.audioCodec = cam.AudioCodec
		}
	}

	if _, ok := parseRTSPAuth(opts.authMode); !ok {
		return fmt.Errorf("invalid --rtsp-auth (use auto|basic|digest)")
	}
	xport, ok := transportFlag(opts.transport)
	if !ok {
		return fmt.Errorf("invalid --rtsp-transport (use tcp|udp)")
	}
	url, err := rtsp.BuildURL(cam)
	if err != nil {
		return err
	}

	ctx, cancel := exec.WithTimeout(context.Background(), opts.timeout)
	defer cancel()

	if opts.path != "" {
		url = appendPath(url, opts.path)
	} else {
		url = appendStream(url, opts.stream)
	}

	ffArgs := []string{}
	if opts.overwrite {
		ffArgs = append(ffArgs, "-y")
	} else {
		ffArgs = append(ffArgs, "-n")
	}

	durStr := fmt.Sprintf("%d", int(opts.duration.Seconds()))
	if opts.duration.Truncate(time.Second) != opts.duration {
		durStr = fmt.Sprintf("%g", opts.duration.Seconds())
	}

	ffArgs = append(ffArgs,
		"-rtsp_transport", xport,
		"-i", url,
		"-t", durStr,
	)

	if opts.audioOnly {
		ffArgs = append(ffArgs, "-vn", "-c:a", opts.audioCodec)
	} else {
		ffArgs = append(ffArgs, "-c:v", "copy")
		if opts.noAudio {
			ffArgs = append(ffArgs, "-an")
		} else {
			if opts.audioCodec == "" {
				ffArgs = append(ffArgs, "-c:a", "aac")
			} else {
				ffArgs = append(ffArgs, "-c:a", opts.audioCodec)
			}
		}
	}
	ffArgs = append(ffArgs, opts.outPath)

	if err := exec.RunFFmpeg(ctx, ffArgs...); err != nil {
		return err
	}

	token := resolveSlackToken(opts.slackToken, cfg)
	ch := resolveSlackChannel(opts.slackChannel, cfg)
	return maybeUploadToSlack(opts.outPath, token, ch, opts.slackMessage, cmd)
}
