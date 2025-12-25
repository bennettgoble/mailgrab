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
Usage:
  mailgrab [OPTIONS]

Application Options:
  -c, --config=      Path to config file [$MAILGRAB_CONFIG]
  -s, --server=      IMAP server hostname [$MAILGRAB_SERVER]
  -p, --port=        IMAP port (default: 993) [$MAILGRAB_PORT]
  -u, --username=    IMAP username [$MAILGRAB_USERNAME]
  -P, --password=    IMAP password [$MAILGRAB_PASSWORD]
  -m, --mailbox=     Mailbox to check (default: Inbox) [$MAILGRAB_MAILBOX]
  -o, --output=      Output directory for attachments [$MAILGRAB_OUTPUT]
      --post-action= Action after processing: none, delete, move (default: none) [$MAILGRAB_POST_ACTION]
      --move-to=     Target folder for move action [$MAILGRAB_MOVE_TO]
      --insecure     Disable TLS verification [$MAILGRAB_INSECURE]
  -v, --verbose      Enable verbose output [$MAILGRAB_VERBOSE]
  -q, --quiet        Suppress non-error output [$MAILGRAB_QUIET]

Help Options:
  -h, --help         Show this help message
```

### Configuration

Configuration can be provided via:
1. Command-line flags
2. Environment variables
3. Config file

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
