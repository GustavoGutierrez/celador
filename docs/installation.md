# Installation

Celador is distributed as a precompiled Go CLI binary.

## Supported release distribution

Current release automation publishes these assets:

- Linux amd64: `celador_X.Y.Z_linux_amd64.tar.gz`
- macOS arm64: `celador_X.Y.Z_darwin_arm64.tar.gz`
- Windows amd64: `celador_X.Y.Z_windows_amd64.zip`

## macOS and Linux with Homebrew

Use the dedicated Homebrew tap:

```bash
brew tap GustavoGutierrez/celador
brew install GustavoGutierrez/celador/celador
```

Homebrew resolves `brew tap GustavoGutierrez/celador` to the repository `GustavoGutierrez/homebrew-celador`.

Upgrade:

```bash
brew update
brew upgrade celador
```

Uninstall:

```bash
brew uninstall celador
brew untap GustavoGutierrez/celador
```

## Windows

Do not install Celador on Windows with Homebrew.

Download the Windows release asset from GitHub Releases instead:

- Releases: <https://github.com/GustavoGutierrez/celador/releases>
- Expected archive name: `celador_X.Y.Z_windows_amd64.zip`

After extracting the archive, place the binary on your `PATH`.

## Build from source

Building from source is the practical fallback for contributors and unsupported release targets.

Requirements:

- Go toolchain matching `go.mod`

Build:

```bash
go build -o celador ./cmd/celador
```

Run locally:

```bash
./celador --help
```

Install into a common local path on Unix-like systems:

```bash
install -m 0755 ./celador /usr/local/bin/celador
```
