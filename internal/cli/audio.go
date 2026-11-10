package cli

import (
	"time"

	"github.com/spf13/cobra"
)

func newAudioCmd() *cobra.Command {
	var cameraName string
	var outPath string
	var duration time.Duration
	var seconds int
	var timeout time.Duration
	var authMode string
	var transport string
	var stream string
	var audioCodec string
	var path string
	var overwrite bool
	var slackChannel string
	var slackToken string
	var slackMessage string

	cmd := &cobra.Command{
		Use:     "audio [camera]",
		Aliases: []string{"record-audio"},
		Short:   "Record an audio-only clip to a file",
		RunE: func(cmd *cobra.Command, args []string) error {
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
				noAudio:      false,
				audioOnly:    true,
				audioCodec:   audioCodec,
				overwrite:    overwrite,
				slackChannel: slackChannel,
				slackToken:   slackToken,
				slackMessage: slackMessage,
			})
		},
	}

	cmd.Flags().StringVar(&cameraName, "camera", "", "Camera name to use")
	cmd.Flags().StringVar(&outPath, "out", "", "Output file (e.g., output_audio.wav)")
	cmd.Flags().DurationVar(&duration, "dur", 10*time.Second, "Recording duration (e.g., 10s)")
	cmd.Flags().IntVarP(&seconds, "seconds", "s", 0, "Recording duration in seconds (alternative to --dur)")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "Timeout for ffmpeg invocation")
	cmd.Flags().StringVar(&authMode, "rtsp-auth", "auto", "RTSP auth mode: auto|basic|digest")
	cmd.Flags().StringVar(&transport, "rtsp-transport", "tcp", "RTSP transport: tcp|udp")
	cmd.Flags().StringVar(&stream, "stream", "", "RTSP path segment (stream1 or stream2); ignored if --path is set")
	cmd.Flags().StringVar(&path, "path", "", "Custom RTSP path (overrides --stream), e.g., /Bfy... from UniFi Protect")
	cmd.Flags().StringVar(&audioCodec, "audio-codec", "", "Audio codec (default copy)")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite output file if it exists")
	addSlackFlags(cmd, &slackChannel, &slackToken, &slackMessage)

	return cmd
}
