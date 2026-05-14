# Installation

## What gets installed

The installer places the `pk3s` executable in your `PATH`.

It does not install Productive K3S Core or Productive K3S Infra bundles up front. Those are resolved on demand when you run a command.

## Unix-like install

Install from the public script:

```bash
curl -fsSL https://raw.githubusercontent.com/jemacchi/productive-k3s-cli/main/install.sh | bash
```

By default the installer writes to:

```text
$HOME/.local/bin/pk3s
```

After installation:

```bash
pk3s version
pk3s help
```

## Windows

The CLI executable is intended to be published as a native Windows binary too.

For this first release, the recommended user-facing flow is:

1. Download the Windows release asset.
2. Put `pk3s.exe` in a directory that is already in `PATH`, or add its directory to `PATH`.
3. Run:

```powershell
pk3s version
pk3s help
```

## Local source install

If you are working from the repository itself:

```bash
make build
./pk3s version
```

## Bundle cache

The CLI stores downloaded bundles in a local cache.

Typical paths:

- Linux/macOS: `~/.cache/pk3s`
- Windows: `%LocalAppData%\pk3s`
