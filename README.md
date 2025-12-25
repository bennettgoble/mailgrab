![mailgrab](./mailgrab.png)

**Mailgrab** is a small CLI tool that checks an IMAP mailbox, downloads image attachments from new emails, and saves them to a specified directory.

## Installation

```bash
go install github.com/bennettgoble/mailgrab@latest
```

Or build from source:

```bash
git clone https://github.com/bennettgoble/mailgrab.git
cd mailgrab
go build
```

## Usage

```bash
mailgrab [options]
```

### Options

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `-c, --config` | `MAILGRAB_CONFIG` | Path to config file |
| `-s, --server` | `MAILGRAB_SERVER` | IMAP server hostname (required) |
| `-p, --port` | `MAILGRAB_PORT` | IMAP port (default: 993) |
| `-u, --username` | `MAILGRAB_USERNAME` | IMAP username (required) |
| `-P, --password` | `MAILGRAB_PASSWORD` | IMAP password (required) |
| `-m, --mailbox` | `MAILGRAB_MAILBOX` | Mailbox to check (default: INBOX) |
| `-o, --output` | `MAILGRAB_OUTPUT` | Output directory for attachments (required) |
| `--post-action` | `MAILGRAB_POST_ACTION` | Action after processing: `none`, `delete`, `move` |
| `--move-to` | `MAILGRAB_MOVE_TO` | Target folder for move action |
| `--insecure` | `MAILGRAB_INSECURE` | Disable TLS verification |
| `-v, --verbose` | `MAILGRAB_VERBOSE` | Enable verbose output |
| `-q, --quiet` | `MAILGRAB_QUIET` | Suppress non-error output |

### Configuration

Configuration can be provided via:
1. Command-line flags (highest priority)
2. Environment variables
3. Config file (lowest priority)

Config file locations (in order of precedence):
1. Path specified via `--config`
2. `./mailgrab.yaml`
3. `~/.config/mailgrab/config.yaml`

#### Example config file

```yaml
server: imap.example.com
port: 993
username: user@example.com
password: your-password
mailbox: INBOX
output: /path/to/photos
post_action: none
# move_to: Archive  # required if post_action is "move"
```

### Examples

```bash
# Using a config file
mailgrab --config mailgrab.yaml

# Using command-line flags
mailgrab -s imap.gmail.com -u user@gmail.com \
  -P "app-password" -o ./photos

# Using environment variables
export MAILGRAB_SERVER=imap.gmail.com
export MAILGRAB_USERNAME=user@gmail.com
export MAILGRAB_PASSWORD=app-password
export MAILGRAB_OUTPUT=./photos
mailgrab

# Verbose output
mailgrab -v --config mailgrab.yaml

# Delete emails after processing
mailgrab --post-action delete --config mailgrab.yaml

# Move emails to Archive folder after processing
mailgrab --post-action move --move-to Archive --config mailgrab.yaml
```
