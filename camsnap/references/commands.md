# Camsnap Command Reference

This reference provides detailed information on all `camsnap` CLI commands and their available flags.

## Global Flags
- `--config <path>`: Path to the configuration file (default: `$XDG_CONFIG_HOME/camsnap/config.yaml`).

## Commands

### `add`
Add a new camera configuration.
- `--name <string>`: Unique name for the camera.
- `--host <string>`: Camera IP or hostname.
- `--user <string>`: RTSP username.
- `--pass <string>`: RTSP password.
- `--port <int>`: RTSP port (default: 554).
- `--protocol <rtsp|rtsps>`: Streaming protocol (default: rtsp).
- `--path <string>`: Custom RTSP path (e.g., for UniFi Protect tokens).
- `--stream <string>`: Stream name/ID (e.g., stream1, stream2).
- `--rtsp-transport <udp|tcp>`: RTSP transport protocol.
- `--rtsp-client <ffmpeg|gortsplib>`: RTSP client implementation.

### `list`
List all configured cameras and their settings.

### `snap`
Capture a single frame (snapshot) from a camera.
- `[camera-name]`: Positional argument for the camera name.
- `--out <path>`: Path to save the image. If omitted, a temporary file is created.
- `--timeout <duration>`: Connection timeout (e.g., 20s).
- `--slack-channel <channel>`: Slack channel or user ID to send the snap to.
- `--slack-message <string>`: Optional message to include with the Slack upload.

### `clip`
Record a short video clip from a camera.
- `[camera-name]`: Positional argument for the camera name.
- `--dur <duration>`: Duration of the clip (e.g., 5s, 10s).
- `--out <path>`: Path to save the video (mp4).
- `--no-audio`: Disable audio recording.
- `--audio-codec <codec>`: Set audio codec (e.g., aac).
- `--slack-channel <channel>`: Slack channel or user ID to send the clip to.
- `--slack-message <string>`: Optional message to include with the Slack upload.

### `watch`
Monitor a camera for motion and trigger actions.
- `[camera-name]`: Positional argument for the camera name.
- `--threshold <float>`: Motion sensitivity (0.0 to 1.0).
- `--cooldown <duration>`: Time to wait between actions.
- `--action <command>`: Shell command to execute when motion is detected.
- `--json`: Output motion events as JSON.

### `discover`
Discover ONVIF-compatible cameras on the local network.
- `--info`: Fetch detailed device information.

### `doctor`
Diagnose connection and environment issues.
- `--probe`: Attempt to probe the RTSP stream and identify specific errors.
- `--rtsp-transport <udp|tcp>`: Force transport for the probe.

### `slack`
Configure Slack integration.
- `set --token <string> --channel <string>`: Store Slack credentials.
- `test`: Verify the Slack connection by sending a test message.
