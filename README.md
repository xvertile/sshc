# sshc

**The SSH Client for Power Users**

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green?style=for-the-badge)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey?style=for-the-badge)]()

> A comprehensive terminal-based SSH client that brings the power of tools like MobaXterm to your command line.

Manage dozens of servers, transfer files, monitor systems, and automate tasks - all without leaving your terminal.

## Why sshc?

- **All-in-one** - SSH, SFTP, port forwarding, and file transfers in a single tool
- **Keyboard-driven** - Vim-style navigation, no mouse required
- **File transfers** - Upload/download files and folders with a built-in remote browser
- **Fast** - Written in Go, starts instantly, minimal resource usage
- **Secure** - Works with your existing SSH config and keys
- **Cross-platform** - Linux, macOS, and Windows support

## Features

| Category | Features |
|----------|----------|
| **Sessions** | Quick connect, connection history, real-time status |
| **Transfers** | SFTP browser, file/folder uploads, recursive downloads |
| **Network** | Local/Remote/Dynamic port forwarding with history |
| **Management** | Add, edit, delete, organize hosts with tags |
| **Search** | Fast fuzzy search across all hosts |

## Installation

**macOS (Homebrew):**
```bash
brew install xvertile/sshc/sshc
```

**Linux/macOS (Script):**
```bash
curl -sSL https://raw.githubusercontent.com/xvertile/sshc/main/install.sh | bash
```

**From source:**
```bash
go install github.com/xvertile/sshc@latest
```

## Quick Start

```bash
# Launch TUI
sshc

# Connect directly
sshc my-server

# File transfers
sshc cp local.txt server:/tmp/     # Upload file
sshc cp server:/var/log/ ./logs/   # Download folder
sshc get server                     # Interactive download
sshc send server                    # Interactive upload
```

## Keybindings

| Key | Action |
|-----|--------|
| `Enter` | Connect to host |
| `a` | Add new host |
| `e` | Edit host |
| `d` | Delete host |
| `t` | Quick file transfer |
| `p` | Port forwarding |
| `/` | Search hosts |
| `?` | Help |
| `q` | Quit |

### File Transfer UI

| Key | Action |
|-----|--------|
| `u/d` | Upload/Download |
| `f/d` | File/Folder |
| `Enter` | Confirm selection |
| `Esc` | Go back |

### Remote Browser

| Key | Action |
|-----|--------|
| `j/k` or arrows | Navigate |
| `Enter` | Select/Open |
| `h` or `Backspace` | Parent directory |
| `/` | Search |
| `.` | Toggle hidden files |
| `~` | Go to home |

## Configuration

sshc works directly with your `~/.ssh/config` file. It adds special comment tags for enhanced functionality while maintaining full SSH compatibility.

```ssh
# Tags: production, web
Host web-prod
    HostName 192.168.1.10
    User deploy
    IdentityFile ~/.ssh/prod_key
```

## License

MIT

---

*sshc: SSH, supercharged.*
