package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steipete/camsnap/internal/config"
	"github.com/steipete/camsnap/internal/slack"
)

func newSlackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slack",
		Short: "Manage Slack configuration",
	}

	cmd.AddCommand(newSlackSetCmd())
	cmd.AddCommand(newSlackTestCmd())

	return cmd
}

func newSlackSetCmd() *cobra.Command {
	var token string
	var channel string

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Store Slack credentials in config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, err := configPathFlag(cmd)
			if err != nil {
				return err
			}
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			if token != "" {
				cfg.Slack.Token = token
			}
			if channel != "" {
				cfg.Slack.DefaultChannel = channel
			}

			if err := config.Save(cfgPath, cfg); err != nil {
				return err
			}

			cmd.Printf("Slack configuration updated in %s\n", cfgPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "Slack API token")
	cmd.Flags().StringVar(&channel, "channel", "", "Default Slack channel or user")

	return cmd
}

func newSlackTestCmd() *cobra.Command {
	var channel string
	var message string

	cmd := &cobra.Command{
		Use:   "test <file>",
		Short: "Test Slack upload with a file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			cfgFlag, err := configPathFlag(cmd)
			if err != nil {
				return err
			}
			cfg, _, err := loadConfig(cfgFlag)
			if err != nil {
				return err
			}

			token := resolveSlackToken("", cfg)
			if channel == "" {
				channel = resolveSlackChannel("", cfg)
			}

			if token == "" {
				return fmt.Errorf("no Slack token found (use SLACK_TOKEN env var or 'camsnap slack set --token')")
			}
			if channel == "" {
				return fmt.Errorf("no Slack channel specified (use --channel or 'camsnap slack set --channel')")
			}

			if message == "" {
				message = "Test upload from camsnap"
			}

			uploader := slack.NewSlackUploader(token)
			chID, err := uploader.ResolveChannel(channel)
			if err != nil {
				return fmt.Errorf("resolving channel: %w", err)
			}

			cmd.Printf("Uploading %s to %s (%s)…\n", filePath, channel, chID)
			resp, err := uploader.UploadFile(filePath, chID, message)
			if err != nil {
				return err
			}

			if len(resp.Files) > 0 {
				cmd.Printf("Success! File ID: %s\n", resp.Files[0].ID)
			} else {
				cmd.Printf("Success!\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&channel, "channel", "", "Slack channel or user (overrides default)")
	cmd.Flags().StringVar(&message, "message", "", "Message to post")

	return cmd
}
