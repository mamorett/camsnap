---
name: camsnap
description: Capture snapshots, record video clips, and monitor motion from RTSP/ONVIF cameras. Use when the user wants to interact with cameras, discover new cameras, set up motion alerts, or send media to Slack.
---

# Camsnap Skill

`camsnap` is a high-performance CLI tool for capturing frames and clips from RTSP cameras. It supports persistence for camera configurations and integrates with Slack for real-time alerting.

## Core Capabilities

### 1. Camera Management
- **`add`**: Configure a new camera with RTSP credentials, transport settings, and stream selection.
- **`list`**: Display all registered cameras and their current configuration.
- **`discover`**: Find ONVIF-compatible cameras on your network.

**Examples:**
```sh
# Add a Tapo camera
camsnap add --name kitchen --host 192.168.0.100 --user admin --pass secret --rtsp-transport udp --stream stream2

# List existing cameras
camsnap list
```

### 2. Media Capture
- **`snap`**: Grab a single frame from a camera.
- **`clip`**: Record a short MP4 clip (e.g., 5-10s).
- **Slack Integration**: Use the `--slack-channel` and `--slack-message` flags to send media directly to Slack.

**Examples:**
```sh
# Capture a snapshot to file
camsnap snap kitchen --out /tmp/snapshot.jpg

# Record a 5s clip and send to Slack
camsnap clip kitchen --dur 5s --slack-channel #security --slack-message "Motion detected!"
```

### 3. Motion Monitoring
- **`watch`**: Use FFmpeg-based scene-change detection to monitor for motion.
- **Actions**: Trigger any shell command when motion is detected. Environment variables `CAMSNAP_CAMERA`, `CAMSNAP_SCORE`, and `CAMSNAP_TIME` are available to the action script.

**Examples:**
```sh
# Monitor kitchen for motion (0.2 threshold) and trigger a snap
camsnap watch kitchen --threshold 0.2 --cooldown 10s --action 'camsnap snap kitchen --slack-channel #alerts'
```

### 4. Diagnostics
- **`doctor`**: Check for `ffmpeg` availability, network reachability, and probe RTSP streams.

---

## Detailed Command Reference

### `add`
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

### `snap`
- `[camera-name]`: Positional argument for the camera name.
- `--out <path>`: Path to save the image. If omitted, a temporary file is created.
- `--timeout <duration>`: Connection timeout (e.g., 20s).
- `--slack-channel <channel>`: Slack channel or user ID.

### `clip`
- `[camera-name]`: Positional argument for the camera name.
- `--dur <duration>`: Duration of the clip (e.g., 5s, 10s).
- `--out <path>`: Path to save the video (mp4).
- `--no-audio`: Disable audio recording.
- `--audio-codec <codec>`: Set audio codec (e.g., aac).
- `--slack-channel <channel>`: Slack channel or user ID.

### `watch`
- `[camera-name]`: Positional argument for the camera name.
- `--threshold <float>`: Motion sensitivity (0.0 to 1.0).
- `--cooldown <duration>`: Time to wait between actions.
- `--action <command>`: Shell command to execute when motion is detected.
- `--action-template <string>`: Template with `{camera}`, `{score}`, `{time}`.
- `--json`: Output motion events as JSON.

### `discover`
- `--info`: Fetch detailed device information via ONVIF.

### `doctor`
- `--probe`: Attempt to probe the RTSP stream.
- `--rtsp-transport <udp|tcp>`: Force transport for the probe.

### `slack`
- `set --token <string> --channel <string>`: Store Slack credentials.
- `test`: Verify the Slack connection.

---

## Workflows

### Setting up a new camera from scratch
1. Run `camsnap discover` to find cameras on the network.
2. Use `camsnap add` to register the camera with its RTSP credentials.
3. Verify the setup with `camsnap snap [camera-name]`.
4. If it fails, run `camsnap doctor --probe` to diagnose.

### Configuring Slack Alerts
1. Run `camsnap slack set --token xoxb-... --channel #general` to store credentials.
2. Test the connection with `camsnap slack test`.
3. Use `--slack-channel` with `snap` or `clip` commands.
