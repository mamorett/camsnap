# Plan: Integrate Slack upload into camsnap

## Goal

Allow `camsnap snap` and `camsnap clip` to optionally post the captured file
directly to a Slack channel or DM, using the three-step upload API already
implemented in `slackup.go`.

---

## Step 1 — Move `slackup.go` into a proper internal package

**Current state:** `slackup.go` lives at the repo root with `package main` and
contains a standalone `main()` function for ad-hoc testing. This conflicts with
`cmd/camsnap/main.go`.

**Actions:**

1. Create directory `internal/slack/`.
2. Copy the contents of `slackup.go` into `internal/slack/slack.go`, changing
   the package declaration to `package slack`.
3. Remove the `main()` function (lines 285–323) from the new file — it was only
   scaffolding. The exported types and methods (`SlackUploader`, `ResolveChannel`,
   `UploadFile`, etc.) remain.
4. Delete `slackup.go` from the repo root.

**Result:** a self-contained `internal/slack` package with no build conflicts.

---

## Step 2 — Extend `ResolveChannel` to support DMs

Currently `ResolveChannel` resolves public/private channels only. To send a DM,
we need to support a `@username` or bare User ID (`U…`) input.

**Actions in `internal/slack/slack.go`:**

1. Add a new `UsersListResponse` struct matching Slack's `users.list` JSON shape
   (fields: `ok`, `error`, `members[].id`, `members[].name`,
   `members[].profile.display_name`, `response_metadata.next_cursor`).
2. Add a `resolveUserID(nameOrID string) (string, error)` method on
   `SlackUploader` that:
   - Returns the input unchanged if it already looks like a User ID (`U…`,
     9–11 chars uppercase alphanumeric).
   - Otherwise paginates through `users.list` matching on `name` or
     `profile.display_name`.
3. Add a `OpenDM(userID string) (string, error)` method that calls
   `conversations.open` with the resolved user ID and returns the resulting
   DM channel ID (`D…`).
4. Update `ResolveChannel` to detect the `@…` prefix, strip it, delegate to
   `resolveUserID` + `OpenDM`, and return the DM channel ID transparently. All
   callers continue to use `ResolveChannel` without needing to know whether the
   destination is a channel or a DM.

**Wire-up rule (clear and simple for callers):**
- `#general` or `general` or `C01234567` → channel
- `@alice` or `alice` or `U01234567` → DM (open conversation → DM channel ID)

---

## Step 3 — Add Slack configuration to the config file

Storing the token and a default channel in the config file avoids repeating them
on every command invocation.

**Actions in `internal/config/config.go`:**

1. Add a top-level `Slack` struct:
   ```go
   type SlackConfig struct {
       Token          string `yaml:"token,omitempty"`
       DefaultChannel string `yaml:"default_channel,omitempty"`
   }
   ```
2. Add a `Slack SlackConfig` field to `Config`.

Existing configs without a `slack:` key will simply have the zero value (empty
strings), so there is no breakage.

---

## Step 4 — Add a shared Slack flag helper

Both `snap` and `clip` need the same three flags. Centralise this to avoid
duplication.

**Actions in `internal/cli/slack.go` (new file):**

```
func addSlackFlags(cmd *cobra.Command)
    Registers --slack-channel, --slack-token, --slack-message on cmd.

func resolveSlackToken(flagToken string, cfg config.Config) string
    Returns the first non-empty value among:
      1. flagToken (--slack-token flag)
      2. os.Getenv("SLACK_TOKEN")
      3. cfg.Slack.Token

func resolveSlackChannel(flagChannel string, cfg config.Config) string
    Returns the first non-empty value among:
      1. flagChannel (--slack-channel flag)
      2. cfg.Slack.DefaultChannel

func maybeUploadToSlack(
    filePath, token, channelArg, comment string,
    cmd *cobra.Command,
) error
    If token == "" or channelArg == "" → no-op (return nil).
    Otherwise:
      1. Instantiate slack.NewSlackUploader(token).
      2. Call uploader.ResolveChannel(channelArg) → channelID.
      3. Call uploader.UploadFile(filePath, channelID, comment).
      4. Print a success line (file ID + share URL if present) via cmd.Printf.
      5. On error, return a wrapped error so the parent command propagates it.
```

---

## Step 5 — Wire Slack upload into `snap`

**Actions in `internal/cli/snap.go`:**

1. Add three new local variables:
   ```go
   var slackChannel string
   var slackToken   string
   var slackMessage string
   ```
2. Call `addSlackFlags(cmd)` at the end of `newSnapCmd` (after existing flag
   registrations).
3. At the end of `RunE`, after the frame is written to `outPath` successfully,
   add:
   ```go
   token  := resolveSlackToken(slackToken, cfg)
   ch     := resolveSlackChannel(slackChannel, cfg)
   if err := maybeUploadToSlack(outPath, token, ch, slackMessage, cmd); err != nil {
       return err
   }
   ```

---

## Step 6 — Wire Slack upload into `clip`

**Actions in `internal/cli/clip.go`:** identical pattern to Step 5.

1. Add the three local variables.
2. Call `addSlackFlags(cmd)`.
3. After `exec.RunFFmpeg` returns `nil`, call `maybeUploadToSlack`.

---

## Step 7 — Update `camsnap add` to accept Slack config

Users should be able to store Slack credentials with `camsnap add` (or via a
future `camsnap slack set` sub-command). For now, the simpler option is to
support setting global Slack config through a dedicated flag on `add` or to
document manual YAML editing. We choose a minimal path: add a
`camsnap slack` sub-command.

**Actions:**

1. Create `internal/cli/slackcmd.go` with `newSlackCmd()` returning a `*cobra.Command`.
   - Sub-command: `camsnap slack set --token xoxb-… --channel #general`
     Loads config, sets `cfg.Slack.Token` / `cfg.Slack.DefaultChannel`, saves.
   - Optionally: `camsnap slack test --file /path/to/file [--channel …]`
     for quick smoke-testing the credentials.
2. Register it in `NewRootCommand` in `root.go`:
   ```go
   cmd.AddCommand(newSlackCmd())
   ```

---

## Step 8 — Update `go.mod` / `go.sum` if needed

The `internal/slack` package uses only stdlib (`bytes`, `encoding/json`, `fmt`,
`io`, `mime`, `net/http`, `net/url`, `os`, `path/filepath`, `strings`) — no new
external dependencies required. No `go.mod` changes are needed.

---

## Step 9 — Add unit tests

**`internal/slack/slack_test.go`:**

- `TestIsSlackID` — table-driven tests for the ID heuristic.
- `TestResolveChannelPassthrough` — verify that a valid channel ID is returned
  as-is without an HTTP round-trip (use `httptest.Server` returning an error to
  confirm no request is made).
- `TestResolveChannelByName` — mock `conversations.list` response, verify name
  lookup.
- `TestResolveUserDM` — mock `users.list` + `conversations.open`, verify `@alice`
  → DM channel ID.

**`internal/cli/slack_test.go`:**

- `TestResolveSlackToken` — verify precedence order (flag → env → config).
- `TestResolveSlackChannel` — verify fallback to config default.
- `TestMaybeUploadToSlackNoOp` — verify returns nil when token or channel empty.

---

## Step 10 — Update documentation

1. **README.md** — add a "Slack integration" section showing:
   ```sh
   # Store credentials once
   camsnap slack set --token xoxb-YOUR-TOKEN --channel "#alerts"

   # Send a snap directly to Slack
   camsnap snap kitchen --slack-channel "#alerts"

   # DM a user
   camsnap clip kitchen --dur 5s --slack-channel "@alice" --slack-message "Motion clip"

   # Or use SLACK_TOKEN env var without storing credentials
   SLACK_TOKEN=xoxb-… camsnap snap kitchen --slack-channel C0123456789
   ```
2. **CHANGELOG.md** — add entry under a new version heading.

---

## File map summary

| Action     | File                              | Notes                                              |
|------------|-----------------------------------|----------------------------------------------------|
| Create     | `internal/slack/slack.go`         | Moved + refactored from `slackup.go`               |
| Create     | `internal/slack/slack_test.go`    | New unit tests                                     |
| Delete     | `slackup.go`                      | Root-level standalone replaced by package above    |
| Modify     | `internal/config/config.go`       | Add `SlackConfig` struct and field to `Config`     |
| Modify     | `internal/config/config_test.go`  | Extend tests to cover round-trip of slack config   |
| Create     | `internal/cli/slack.go`           | Shared flag helpers + `maybeUploadToSlack`         |
| Create     | `internal/cli/slack_test.go`      | Unit tests for helpers                             |
| Create     | `internal/cli/slackcmd.go`        | `camsnap slack set` / `camsnap slack test`         |
| Modify     | `internal/cli/snap.go`            | Add Slack flags + post-capture upload call         |
| Modify     | `internal/cli/clip.go`            | Add Slack flags + post-capture upload call         |
| Modify     | `internal/cli/root.go`            | Register `newSlackCmd()`                           |
| Modify     | `README.md`                       | Slack integration section                          |
| Modify     | `CHANGELOG.md`                    | Release note                                       |

---

## Token / credential precedence (enforced by `resolveSlackToken`)

```
1. --slack-token flag       (per-invocation override)
2. SLACK_TOKEN env var      (CI / shell profile)
3. cfg.Slack.Token          (stored by `camsnap slack set`)
```

If none is set, Slack upload is silently skipped (no error) unless the user
explicitly passed `--slack-channel`, in which case an error is returned:
"slack-channel specified but no token found (use --slack-token or SLACK_TOKEN)".

---

## Implementation order (suggested)

1. Step 1 (move package) — unblocks all others.
2. Step 3 (config struct) — needed by Steps 4–7.
3. Steps 2 + 4 (extend ResolveChannel, create cli/slack.go helpers).
4. Steps 5 + 6 (wire into snap/clip).
5. Step 7 (slack sub-command).
6. Steps 9 + 10 (tests and docs) — last, after behaviour is stable.
