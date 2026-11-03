# Install Kirsch

Kirsch is a terminal program. This page covers the supported platforms, what
Kirsch needs on the machine, the ways to install it, how to set up an API key,
and how to check the install with `kirsch doctor`.

> **What this page covers.** Installing the `kirsch` program and checking that
> it works. It does not cover choosing a model provider or changing settings.
> Those are in [configuration.md](configuration.md).

---

## Supported platforms

| Operating system | Architectures |
|---|---|
| macOS | amd64, arm64 |
| Linux | amd64, arm64 |

Windows is not supported. Kirsch stops a running command by stopping its
process group, and Windows has no Unix process groups.

## What Kirsch needs on the machine

### Git

Kirsch works on a Git repository. Without `--workspace`, Kirsch finds the
workspace by asking `git` for the repository that encloses the folder you start
it in. So `git` must be on your `PATH`, and you must start Kirsch inside a
repository. If either is missing, Kirsch does not start, and `kirsch doctor`
reports a failure.

With `--workspace <path>`, Kirsch uses that folder and does not ask `git` for
anything. If the folder is not a Git repository, Kirsch still starts, but its
Git features are off.

### ripgrep (optional)

Kirsch's code search uses `rg` (ripgrep) when `rg` is on your `PATH`. Without
`rg`, Kirsch uses its own built-in search, and search still works. Kirsch does
not stop for the lack of `rg`.

`kirsch doctor` reports the difference. Without `rg`, it prints these warnings:

    warn  rg: not found on PATH; search_code uses the pure-Go fallback
    warn  backend: search_code uses pure-go (rg is not on PATH)

A warning does not make `kirsch doctor` fail. To use ripgrep, install it with
your system's package manager.

## Install from a release archive

Release archives are on the
[releases page](https://github.com/djm56/kirsch/releases).

Each release has a `.tar.gz` archive for every supported platform, and a
checksum file for the whole release. The names follow this pattern, where
`<version>` is the release number without the leading `v`, such as `0.1.0`:

- `kirsch_<version>_<os>_<arch>.tar.gz`, where `<os>` is `darwin` or `linux`
  and `<arch>` is `amd64` or `arm64`
- `kirsch_<version>_checksums.txt`

The steps below use version `0.1.0` on macOS with Apple silicon. Change the
variables at the top for your release and platform.

```bash
VERSION=0.1.0
OS=darwin
ARCH=arm64
ARCHIVE="kirsch_${VERSION}_${OS}_${ARCH}.tar.gz"

curl -fLO "https://github.com/djm56/kirsch/releases/download/v${VERSION}/${ARCHIVE}"
curl -fLO "https://github.com/djm56/kirsch/releases/download/v${VERSION}/kirsch_${VERSION}_checksums.txt"
```

Check the archive against the checksum file before you unpack it. The checksum
file lists every platform, so the command selects the one line for your archive
and passes it to the checker.

On macOS, use `shasum`:

```bash
grep -F "${ARCHIVE}" "kirsch_${VERSION}_checksums.txt" | shasum -a 256 -c -
```

On Linux, use `sha256sum`:

```bash
grep -F "${ARCHIVE}" "kirsch_${VERSION}_checksums.txt" | sha256sum -c -
```

Each command prints a line ending in `OK` when the archive matches. A line
ending in `FAILED` means the download is damaged or incomplete. Download it
again, and do not run it.

Unpack the archive. It contains the `kirsch` program, and may also hold
licence and readme files:

```bash
tar -xzf "${ARCHIVE}"
```

Move `kirsch` onto your `PATH`. Either of these works:

```bash
sudo mkdir -p /usr/local/bin && sudo mv kirsch /usr/local/bin/
```

```bash
mkdir -p ~/.local/bin
mv kirsch ~/.local/bin/
```

If you use `~/.local/bin`, make sure that folder is on your `PATH`.

Then check the version:

```bash
kirsch --version
```

A release binary prints the release version, for example:

```text
kirsch 0.1.0
```

## Install with `go install`

`go install` builds Kirsch from its source on the module proxy. You need Go
1.26.10 or later, the version named in the module's `go` line. Run `go version`
to see what you have. If your Go is older, recent Go releases fetch the
required toolchain automatically. They do this unless `GOTOOLCHAIN` is set to
`local`.

Install the latest version:

```bash
go install github.com/djm56/kirsch/cmd/kirsch@latest
```

Or install one release, by tag:

```bash
go install github.com/djm56/kirsch/cmd/kirsch@v0.1.0
```

The binary goes to `$(go env GOBIN)` when `GOBIN` is set. Otherwise it goes to
`$(go env GOPATH)/bin`. Make sure that folder is on your `PATH`.

The version text depends on what you built from:

| You installed | `kirsch --version` prints |
|---|---|
| A release tag, such as `@v0.1.0`, or `@latest` once a release exists | The tag without its leading `v`, such as `kirsch 0.1.0` |
| Anything else, such as a branch (`@main`) or a commit | `kirsch 0.1.0-dev` |

Go records a commit as a pseudo-version, which is not a release tag, so a
commit build prints `kirsch 0.1.0-dev`.

## Build from source

Build from a clone of the repository. You need Go, as above, and `git`.

```bash
git clone https://github.com/djm56/kirsch.git
cd kirsch
bash scripts/build.sh
./bin/kirsch --version
```

The binary is written to `bin/kirsch` inside the checkout. The script stamps a
version into the binary:

- `bash scripts/build.sh 1.2.3` stamps `1.2.3`. A leading `v` is removed, so
  `v1.2.3` also stamps `1.2.3`.
- `bash scripts/build.sh` with no argument stamps the version only when the
  checkout is exactly on a `v` release tag and has no uncommitted changes.
  Otherwise the binary reports `kirsch 0.1.0-dev`.

## Set up your API key

Kirsch sends model requests through a provider. The default provider is
`opencode`, and it reads its key from the environment variable
`OPENCODE_API_KEY`. The built-in `anthropic` provider reads
`ANTHROPIC_API_KEY`.

Export the key in your shell. Replace the placeholder with your own key:

```bash
export OPENCODE_API_KEY="<your-opencode-key>"
```

Add the line to your shell's startup file if you want it set in every session.

Kirsch also accepts the same name with a `KIRSCH_` prefix, such as
`KIRSCH_OPENCODE_API_KEY`. It checks the prefixed name first. If both are set,
the prefixed one is used. An empty variable counts as unset. The
[secrets rules](configuration.md#secrets) explain the lookup order.

**Never put an API key in a config file.** Kirsch refuses to start when a
config file holds a key whose name looks like a credential. To use another
provider, see [configuration.md](configuration.md).

## Check the install with `kirsch doctor`

Run `kirsch doctor` inside your project folder:

```bash
cd /path/to/your/project
kirsch doctor
```

To check a folder other than the current one, pass `--workspace <path>`.

Doctor checks what Kirsch depends on, such as `git`, `rg`, your project and
settings, your API key, the model, the session lock, and the folder where
Kirsch saves sessions. It creates no files and makes no network calls. It never
sends a prompt to a model.

The report starts with the version line. Each check then gets one line with its
status (`ok`, `warn`, or `fail`), the check name, and a detail. The last line
counts the results.

This is an example. Your lines and paths depend on your machine:

```text
kirsch 0.1.0
ok  git: found at /usr/bin/git
ok  rg: found at /usr/local/bin/rg
ok  repository: /home/you/project
ok  config: settings valid
warn  key: endpoint opencode: no key found (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY); kirsch starts but cannot send a prompt
ok  backend: search_code uses ripgrep
ok  model: endpoint opencode, model minimax-m3: context window 128000 tokens
ok  context: AGENTS.md, 6854 bytes
ok  state: /home/you/.local/share/kirsch/sessions/3f9a1c07b2d4 exists and is writable
ok  lock: free
doctor: 9 ok, 1 warn, 0 fail
```

The first time you run doctor, before Kirsch has run in that project, the
`state` line says the folder does not exist yet and that Kirsch will create it,
and the `lock` line says `free (no lock file yet)`. Each is `ok`.

Doctor exits with status 0 when no check fails, and with status 1 when at
least one check fails. A warning does not cause a failure. A `fail` line names
the problem, and it is the one to fix before you use Kirsch.
