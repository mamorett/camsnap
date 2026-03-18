package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var slackAPIBase = "https://slack.com/api"

// UploadURLResponse is the response from files.getUploadURLExternal
type UploadURLResponse struct {
	OK        bool   `json:"ok"`
	UploadURL string `json:"upload_url"`
	FileID    string `json:"file_id"`
	Error     string `json:"error,omitempty"`
}

// CompleteUploadResponse is the response from files.completeUploadExternal
type CompleteUploadResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Files []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"files,omitempty"`
}

// ConversationsListResponse is the response from conversations.list
type ConversationsListResponse struct {
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
	Channels []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"channels"`
	ResponseMetadata struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
}

// UsersListResponse is the response from users.list
type UsersListResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
	Members []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Profile struct {
			DisplayName string `json:"display_name"`
		} `json:"profile"`
	} `json:"members"`
	ResponseMetadata struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
}

// ConversationsOpenResponse is the response from conversations.open
type ConversationsOpenResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
	Channel struct {
		ID string `json:"id"`
	} `json:"channel"`
}

// SlackUploader holds the configuration for uploading files to Slack.
type SlackUploader struct {
	Token   string
	APIBase string
}

// NewSlackUploader creates a new SlackUploader with the given token.
func NewSlackUploader(token string) *SlackUploader {
	return &SlackUploader{Token: token, APIBase: slackAPIBase}
}

func (s *SlackUploader) apiURL(path string) string {
	base := s.APIBase
	if base == "" {
		base = slackAPIBase
	}
	return base + path
}

// ResolveChannel accepts either a channel ID (e.g. "C01234567"), a channel
// name with or without the leading "#" (e.g. "#general" or "general"), or a
// user name/ID with or without the leading "@" (e.g. "@alice" or "alice").
// It always returns the channel ID (C... or D...) required by the upload API.
func (s *SlackUploader) ResolveChannel(nameOrID string) (string, error) {
	if strings.HasPrefix(nameOrID, "@") {
		userID, err := s.resolveUserID(strings.TrimPrefix(nameOrID, "@"))
		if err != nil {
			return "", err
		}
		return s.OpenDM(userID)
	}

	// Strip optional leading '#'
	nameOrID = strings.TrimPrefix(nameOrID, "#")

	// Already looks like an ID — Slack IDs start with C, G, D, W, U…
	if len(nameOrID) > 1 && nameOrID[0] != strings.ToLower(nameOrID)[0] {
		// If it's a User ID (U...), we need to open a DM to get a channel ID (D...)
		if strings.HasPrefix(nameOrID, "U") {
			return s.OpenDM(nameOrID)
		}
		return nameOrID, nil
	}
	// Simple heuristic: IDs are uppercase alphanumeric and typically 9-11 chars
	if isSlackID(nameOrID) {
		if strings.HasPrefix(nameOrID, "U") {
			return s.OpenDM(nameOrID)
		}
		return nameOrID, nil
	}

	// Paginate through conversations.list to find the matching name
	cursor := ""
	for {
		apiURL := fmt.Sprintf(
			"%s/conversations.list?limit=200&exclude_archived=true&types=public_channel,private_channel",
			s.apiURL(""),
		)
		if cursor != "" {
			apiURL += "&cursor=" + url.QueryEscape(cursor)
		}

		req, err := http.NewRequest(http.MethodGet, apiURL, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+s.Token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		var result ConversationsListResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return "", err
		}
		if !result.OK {
			return "", fmt.Errorf("slack API error: %s", result.Error)
		}

		for _, ch := range result.Channels {
			if ch.Name == nameOrID {
				return ch.ID, nil
			}
		}

		cursor = result.ResponseMetadata.NextCursor
		if cursor == "" {
			break
		}
	}

	// If not found in channels, try users
	userID, err := s.resolveUserID(nameOrID)
	if err == nil {
		return s.OpenDM(userID)
	}

	return "", fmt.Errorf("channel or user %q not found (make sure the bot is a member)", nameOrID)
}

// resolveUserID resolves a username or display name to a User ID (U...).
func (s *SlackUploader) resolveUserID(nameOrID string) (string, error) {
	if isSlackID(nameOrID) && strings.HasPrefix(nameOrID, "U") {
		return nameOrID, nil
	}

	cursor := ""
	for {
		apiURL := s.apiURL("/users.list?limit=200")
		if cursor != "" {
			apiURL += "&cursor=" + url.QueryEscape(cursor)
		}

		req, err := http.NewRequest(http.MethodGet, apiURL, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+s.Token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		var result UsersListResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return "", err
		}
		if !result.OK {
			return "", fmt.Errorf("slack API error: %s", result.Error)
		}

		for _, m := range result.Members {
			if m.Name == nameOrID || m.Profile.DisplayName == nameOrID {
				return m.ID, nil
			}
		}

		cursor = result.ResponseMetadata.NextCursor
		if cursor == "" {
			break
		}
	}

	return "", fmt.Errorf("user %q not found", nameOrID)
}

// OpenDM opens a DM conversation with a user and returns the DM channel ID (D...).
func (s *SlackUploader) OpenDM(userID string) (string, error) {
	type requestBody struct {
		Users string `json:"users"`
	}
	rb := requestBody{Users: userID}
	bodyBytes, err := json.Marshal(rb)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, s.apiURL("/conversations.open"), bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result ConversationsOpenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.OK {
		return "", fmt.Errorf("slack API error: %s", result.Error)
	}

	return result.Channel.ID, nil
}

// isSlackID returns true for strings that look like Slack resource IDs
// (all-uppercase alphanumeric, 9-11 chars, starting with a letter).
func isSlackID(s string) bool {
	if len(s) < 9 || len(s) > 11 {
		return false
	}
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// UploadFile uploads a file to a Slack channel using the new three-step API.
// channelID can be empty to upload without sharing to a channel.
func (s *SlackUploader) UploadFile(filePath, channelID, initialComment string) (*CompleteUploadResponse, error) {
	// --- Step 0: Read the file info ---
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	fileInfo, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	fileName := filepath.Base(filePath)
	fileSize := fileInfo.Size()
	mimeType := mime.TypeByExtension(filepath.Ext(filePath))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// --- Step 1: Get upload URL ---
	uploadResp, err := s.getUploadURL(fileName, fileSize, mimeType)
	if err != nil {
		return nil, fmt.Errorf("getting upload URL (step 1/3): %w", err)
	}

	// --- Step 2: Upload the file content to the provided URL ---
	if err := s.uploadContent(uploadResp.UploadURL, f, fileSize, mimeType); err != nil {
		return nil, fmt.Errorf("uploading file content (step 2/3): %w", err)
	}

	// --- Step 3: Complete the upload ---
	completeResp, err := s.completeUpload(uploadResp.FileID, channelID, initialComment)
	if err != nil {
		return nil, fmt.Errorf("completing upload (step 3/3): %w", err)
	}

	return completeResp, nil
}

// getUploadURL calls files.getUploadURLExternal to obtain a pre-signed upload URL.
func (s *SlackUploader) getUploadURL(fileName string, fileSize int64, mimeType string) (*UploadURLResponse, error) {
	v := url.Values{}
	v.Set("filename", fileName)
	v.Set("length", fmt.Sprintf("%d", fileSize))

	apiURL := s.apiURL("/files.getUploadURLExternal") + "?" + v.Encode()

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result UploadURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if !result.OK {
		return nil, fmt.Errorf("slack API error: %s", result.Error)
	}
	return &result, nil
}

// uploadContent performs a raw POST of the file bytes to the pre-signed upload URL.
func (s *SlackUploader) uploadContent(uploadURL string, data io.Reader, size int64, mimeType string) error {
	req, err := http.NewRequest(http.MethodPost, uploadURL, data)
	if err != nil {
		return err
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", mimeType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// completeUpload calls files.completeUploadExternal to finalise the upload and
// optionally share the file to a channel.
func (s *SlackUploader) completeUpload(fileID, channelID, initialComment string) (*CompleteUploadResponse, error) {
	type fileEntry struct {
		ID string `json:"id"`
	}
	type requestBody struct {
		Files          []fileEntry `json:"files"`
		ChannelID      string      `json:"channel_id,omitempty"`
		InitialComment string      `json:"initial_comment,omitempty"`
	}

	body := requestBody{
		Files:          []fileEntry{{ID: fileID}},
		ChannelID:      channelID,
		InitialComment: initialComment,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, s.apiURL("/files.completeUploadExternal"), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result CompleteUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if !result.OK {
		return nil, fmt.Errorf("slack API error: %s", result.Error)
	}
	return &result, nil
}
