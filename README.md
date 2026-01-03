# sshc

A terminal SSH client with file transfers, port forwarding, and host management.

<video src="images/initial-showcase.mp4" autoplay loop muted playsinline></video>

---

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/xvertile/sshc/main/install/install.sh | bash
```

or via Homebrew:

```bash
brew tap xvertile/sshc
brew install sshc
```

---

## Usage

```bash
sshc                    # interactive mode
sshc <host>             # direct connection
sshc transfer <host>    # file transfer
sshc forward <host>     # port forwarding
```

### Keys

```
j/k, arrows    navigate
enter          connect
a              add host
e              edit host
d              delete host
f              port forward
t              transfer
/              search
q              quit
```

---

## Features

**Connection Management**
- TUI for browsing and connecting to hosts
- Real-time connectivity status
- Connection history

**File Transfer**
- Upload/download via SCP
- Recursive directory transfers
- Native file picker integration
- Remote file browser

**Port Forwarding**
- Local (-L), remote (-R), dynamic (-D)
- Saved configurations

**Configuration**
- Works with ~/.ssh/config
- Include directive support
- ProxyJump for bastion hosts

---

## Configuration

Uses `~/.ssh/config` by default. Custom configs:

```bash
sshc -c /path/to/config
```

Data stored in `~/.config/sshc/`

---

## Requirements

- OpenSSH client
- Go 1.23+ (building from source)
- Linux file picker: zenity or kdialog

---

## Credits

Continuation of [sshm](https://github.com/Gu1llaum-3/sshm) by Guillaume.

## License

MIT
