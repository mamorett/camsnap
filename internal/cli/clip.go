package cli

import (
	"time"

	"github.com/spf13/cobra"
)

func newClipCmd() *cobra.Command {
	var cameraName string
	var outPath string
	var duration time.Duration
	var seconds int
	var timeout time.Duration
	var authMode string
	var transport string
	var stream string
	var noAudio bool
	var audioOnly bool
	var audioCodec string
	var path string
	var overwrite bool
	var slackChannel string
	var slackToken string
	var slackMessage string

	cmd := &cobra.Command{
		Use:   "clip",
		Short: "Record a short clip to a file",
		RunE: func(cmd *cobra.Command, args []string) error {
			// For standard video clip, default overwrite to true for backward compatibility
			// unless explicitly set to false or when --audio-only is requested.
			effOverwrite := overwrite
			if !cmd.Flags().Changed("overwrite") {
				if !audioOnly {
					effOverwrite = true
				}
			}

			return executeRecord(cmd, args, recordOptions{
				cameraName:   cameraName,
				outPath:      outPath,
				duration:     duration,
				seconds:      seconds,
				timeout:      timeout,
				authMode:     authMode,
				transport:    transport,
				stream:       stream,
				path:         path,
				noAudio:      noAudio,
				audioOnly:    audioOnly,
				audioCodec:   audioCodec,
				overwrite:    effOverwrite,
				slackChannel: slackChannel,
				slackToken:   slackToken,
				slackMessage: slackMessage,
			})
		},
	}

	cmd.Flags().StringVar(&cameraName, "camera", "", "Camera name to use")
	cmd.Flags().StringVar(&outPath, "out", "", "Output file (e.g., clip.mp4)")
	cmd.Flags().DurationVar(&duration, "dur", 10*time.Second, "Clip duration (e.g., 10s)")
	cmd.Flags().IntVarP(&seconds, "seconds", "s", 0, "Clip duration in seconds (alternative to --dur)")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "Timeout for ffmpeg invocation")
	cmd.Flags().StringVar(&authMode, "rtsp-auth", "auto", "RTSP auth mode: auto|basic|digest")
	cmd.Flags().StringVar(&transport, "rtsp-transport", "tcp", "RTSP transport: tcp|udp")
	cmd.Flags().StringVar(&stream, "stream", "", "RTSP path segment (stream1 or stream2); ignored if --path is set")
	cmd.Flags().StringVar(&path, "path", "", "Custom RTSP path (overrides --stream), e.g., /Bfy... from UniFi Protect")
	cmd.Flags().BoolVar(&noAudio, "no-audio", false, "Drop audio track")
	cmd.Flags().BoolVar(&audioOnly, "audio-only", false, "Record audio only (drop video track)")
	cmd.Flags().StringVar(&audioCodec, "audio-codec", "", "Audio codec (default aac for video clip, copy for audio-only); ignored if --no-audio")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite output file if it exists")
	addSlackFlags(cmd, &slackChannel, &slackToken, &slackMessage)

	return cmd
}
