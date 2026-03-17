package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/steipete/camsnap/internal/config"
	"github.com/steipete/camsnap/internal/slack"
)

func addSlackFlags(cmd *cobra.Command, channel, token, message *string) {
	cmd.Flags().StringVar(channel, "slack-channel", "", "Slack channel or user to upload to (e.g. #general, @alice)")
	cmd.Flags().StringVar(token, "slack-token", "", "Slack API token (overrides config and SLACK_TOKEN env var)")
	cmd.Flags().StringVar(message, "slack-message", "", "Optional message to post with the file")
}

func resolveSlackToken(flagToken string, cfg config.Config) string {
	if flagToken != "" {
		return flagToken
	}
	if envToken := os.Getenv("SLACK_TOKEN"); envToken != "" {
		return envToken
	}
	return cfg.Slack.Token
}

func resolveSlackChannel(flagChannel string, cfg config.Config) string {
	if flagChannel != "" {
		return flagChannel
	}
	return cfg.Slack.DefaultChannel
}

func maybeUploadToSlack(
	filePath, token, channelArg, comment string,
	cmd *cobra.Command,
) error {
	if channelArg == "" {
		return nil
	}
	if token == "" {
		return fmt.Errorf("slack-channel specified but no token found (use --slack-token or SLACK_TOKEN environment variable)")
	}

	cmd.Printf("Uploading %s to Slack (%s)…\n", filePath, channelArg)

	uploader := slack.NewSlackUploader(token)
	channelID, err := uploader.ResolveChannel(channelArg)
	if err != nil {
		return fmt.Errorf("resolving slack channel: %w", err)
	}

	resp, err := uploader.UploadFile(filePath, channelID, comment)
	if err != nil {
		return fmt.Errorf("uploading to slack: %w", err)
	}

	if len(resp.Files) > 0 {
		cmd.Printf("Slack upload complete! File ID: %s\n", resp.Files[0].ID)
	} else {
		cmd.Printf("Slack upload complete!\n")
	}

	return nil
}
