# pop3t

A command line tool for managing a POP3 mailbox: list, read, delete, forward, and filter
messages, and send mail over SMTP. Messages are classified by language using
[lingua](https://github.com/pemistahl/lingua-go) and, optionally, as spam or malicious using
[jev](https://github.com/leftmike/gjevt).

## Install

```
go install github.com/leftmike/pop3t@latest
```

## Usage

```
pop3t <cmd> [flags] [args]
```

| Command | Description |
| --- | --- |
| `list` | List all messages with id, language, and subject. |
| `get [-format brief\|normal\|full\|raw\|parts] <id\|range>...` | Display messages. |
| `delete <id\|range>...` | Delete messages, saving them to the archive directory if one is set. |
| `forward <addr> <id\|range>...` | Forward messages as attachments. |
| `filter <filters>... [delete] [forward=<addr>]` | Show, and optionally delete or forward, matching messages. |
| `send [-subject <subject>] <addr>...` | Send a message; the body is read from stdin. |
| `test <file>...` | Classify `.eml` files. |
| `help` | Show help. |

Ids are message numbers from `list`; a range is `lo-hi`.

### Filters

- `<lang>` matches messages in that language, e.g. `english`. `^<lang>` matches messages not
  in that language. Language filters must be all positive or all negative.
- `spam` and `malicious` match messages that jev classifies as such; `^spam` and `^malicious`
  match messages that it doesn't. These require a jev API key.

For example, `pop3t filter ^english delete` deletes every message not in English, and
`pop3t filter spam forward=me@example.com` forwards spam.

## Configuration

Settings are read from the first of these files that exists, or from `-config <file>`:

- `~/.pop3t/pop3t.hcl` (`~/pop3t/pop3t.hcl` on Windows)
- `~/.pop3t.hcl`
- `./pop3t.hcl`

```hcl
host     = "mail.example.com"
user     = "me@example.com"
password = "secret"
archive  = "/home/me/mail-archive"

pop3 {
  port   = 995
  no_tls = false
}

smtp {
  port   = 587
  no_tls = false
}

jev {
  api_key  = "..."
  base_url = "..."
  model    = "..."
}
```

`host`, `user`, and `password` apply to both POP3 and SMTP unless they are overridden in the
`pop3` or `smtp` block.

POP3 uses TLS on port 995 by default, or plain text on port 110 with `no_tls`. SMTP uses
STARTTLS on port 587 by default, implicit TLS when the port is 465, or plain text on port 25
with `no_tls`.

Every setting can also be given as a flag, e.g. `-pop3-host`, `-smtp-port`, `-jev-model`, or
`-archive`; run `pop3t <cmd> -h` for the full list. To keep secrets out of `ps` and shell
history, passwords can come from the environment instead: `POP3T_PASSWORD`,
`POP3T_POP3_PASSWORD`, `POP3T_SMTP_PASSWORD`, and `POP3T_JEV_API_KEY`. Flags take precedence
over the environment, which takes precedence over the config file.
