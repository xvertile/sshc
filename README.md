<p align="center">
  <h1 align="center">sshc</h1>
  <p align="center">A modern, interactive terminal UI for managing SSH connections</p>
</p>

<p align="center">
  <a href="https://github.com/xvertile/sshc/releases"><img src="https://img.shields.io/github/v/release/xvertile/sshc?style=flat-square&color=blue" alt="Release"></a>
  <a href="https://github.com/xvertile/sshc/blob/main/LICENSE"><img src="https://img.shields.io/github/license/xvertile/sshc?style=flat-square&color=lightgray" alt="License"></a>
  <a href="https://goreportcard.com/report/github.com/xvertile/sshc"><img src="https://goreportcard.com/badge/github.com/xvertile/sshc?style=flat-square" alt="Go Report Card"></a>
  <img src="https://img.shields.io/badge/go-1.23+-00ADD8?style=flat-square" alt="Go Version">
</p>

<br>

<p align="center">
  <img src="images/menu.png" alt="sshc" width="700">
</p>

<br>

SSHC transforms your `~/.ssh/config` into a searchable, navigable interface — letting you connect to servers, transfer files, and manage hosts without memorizing hostnames or typing lengthy commands.

Built with Go and the [Charm](https://charm.sh) ecosystem for a fast, responsive experience that feels native to the terminal.

---

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/xvertile/sshc/main/install/install.sh | bash
```

**Homebrew**

```bash
brew tap xvertile/sshc && brew install sshc
```

**From source**

```bash
go install github.com/xvertile/sshc@latest
```

<br>

<p align="center">
  <img src="images/showcase.gif" alt="showcase" width="700">
</p>
