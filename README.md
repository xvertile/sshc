# SSHC - SSH Client

A feature-rich SSH client for the terminal, built as a continuation of [SSHM](https://github.com/Gu1llaum-3/sshm).

SSHC extends the original SSH manager with additional client capabilities including file transfers, remote browsing, and more.

## Features

### Connection Management
- Interactive TUI for browsing and connecting to SSH hosts
- Direct CLI connection with `sshc <host>`
- Real-time connectivity status indicators
- Connection history with last login tracking
- Support for custom SSH config files

### File Transfer
- Upload and download files via SCP
- Transfer entire directories recursively
- Native file picker integration (macOS, Linux)
- Remote file browser for selecting transfer targets
- Transfer history tracking

### Port Forwarding
- Local port forwarding (-L)
- Remote port forwarding (-R)
- Dynamic SOCKS proxy (-D)
- Saved forwarding configurations for quick reuse

### Configuration
- Works directly with ~/.ssh/config
- Full SSH Include directive support
- Add, edit, move, and delete host configurations
- Tag-based organization
- ProxyJump support for bastion hosts
- Any SSH option can be configured

## Installation

### Quick Install (Recommended)
```bash
curl -fsSL https://raw.githubusercontent.com/xvertile/sshc/main/install/install.sh | bash
```

### Homebrew (macOS/Linux)
```bash
brew tap xvertile/sshc
brew install sshc
```

To upgrade:
```bash
brew update && brew upgrade sshc
```

### From Source
```bash
git clone https://github.com/xvertile/sshc.git
cd sshc
go build -o sshc .
sudo mv sshc /usr/local/bin/
```

### Binary Releases

Download the latest release for your platform from the [releases page](https://github.com/xvertile/sshc/releases).

## Usage

### Interactive Mode

Launch without arguments to open the TUI:
```bash
sshc
```

Navigation:
- `j/k` or arrows - navigate host list
- `Enter` - connect to selected host
- `a` - add new host
- `e` - edit host
- `d` - delete host
- `m` - move host to another config file
- `f` - port forwarding setup
- `t` - file transfer
- `/` - search hosts
- `q` - quit

### Direct Connection
```bash
sshc production-server
sshc db-staging -c ~/work/ssh_config
```

### File Transfer
```bash
# Interactive transfer UI
sshc transfer myserver

# Or press 't' in the TUI while a host is selected
```

The transfer interface provides:
- Choice between upload and download
- Native file picker for local files
- Remote browser for selecting destination/source
- Progress indication during transfer

### Port Forwarding
```bash
# Interactive forwarding setup
sshc forward myserver

# Or press 'f' in the TUI
```

### Host Management
```bash
# Add a new host
sshc add

# Edit existing host
sshc edit myserver

# Move host to different config file
sshc move myserver
```

## Configuration

SSHC uses your existing SSH configuration at `~/.ssh/config`. Custom config files can be specified with `-c`:
```bash
sshc -c /path/to/config
```

### SSH Include Support

Organize configurations across multiple files:
```ssh
# ~/.ssh/config
Include ~/.ssh/conf.d/*
Include work-servers.conf

Host personal
    HostName personal.example.com
    User me
```

### Key Bindings

Custom key bindings can be configured in `~/.config/sshc/config.json`:
```json
{
  "key_bindings": {
    "quit_keys": ["q", "ctrl+c"],
    "disable_esc_quit": true
  }
}
```

### Data Storage

- Config: `~/.config/sshc/`
- Backups: `~/.config/sshc/backups/`
- History: `~/.config/sshc/history.json`

## Requirements

- Go 1.23+ (for building from source)
- OpenSSH client
- For native file picker on Linux: zenity or kdialog

## Credits

SSHC is a continuation of [SSHM](https://github.com/Gu1llaum-3/sshm) by [Guillaume](https://github.com/Gu1llaum-3). The original project provided the foundation for host management and the TUI interface.

## License

MIT License - see [LICENSE](LICENSE) for details.