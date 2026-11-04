# Security Policy

## Supported versions

Only the latest release of Kirsch receives security fixes. If you run an older
release, upgrade to the latest one before you report a problem with it.

## Reporting a vulnerability

Report security problems privately. Do not open a public issue or pull request
for them.

Use GitHub's private vulnerability reporting form:

https://github.com/djm56/kirsch/security/advisories/new

You can reach the form from the Security tab of the repository. Include the
version you ran, your operating system, and the steps that reproduce the
problem.

We aim to reply within 7 days.

## Threat model

Kirsch is a terminal coding assistant. It reads the repository you open, sends
your prompts and tool results to the model provider you configure, and can
change files and run commands in that repository. The things worth protecting
are:

- the files in your workspace, including `.env` files and private keys;
- your API keys and tokens;
- your user account, which any command Kirsch runs would have.

The controls below are the ones Kirsch relies on. Some of them have known
limits, which the next section lists.

- **Changes need your approval.** The file tools apply a patch only after you
  approve it.
- **Commands need a policy decision.** By default, a command runs only if it is
  on the built-in allowlist, a session grant you gave covers it, or you approve
  it. The approval prompt shows the full command. Kirsch runs the command
  directly, without a shell.
- **Paths stay inside the workspace.** Kirsch refuses paths outside the
  workspace root, including paths that reach outside through a symlink. It also
  refuses `.env` files, key files such as `*.pem`, and the `.git/` and `.kirsch/`
  directories. The match ignores case.
- **Commands do not receive secret-named variables.** Before a command runs,
  Kirsch removes environment variables whose names end in `_TOKEN`, `_KEY` or
  `_SECRET`, or start with `AWS_`. It removes them even when they are on the
  passthrough list.
- **Credentials stay out of config files.** Kirsch refuses to load a config file
  that holds a credential-shaped key. A project's config file can set only the
  list of context files. Its other settings, including the debug log, the
  providers and the policy, are ignored.
- **Remote endpoints are checked.** A base URL that uses plain `http` must point
  at a loopback address. A base URL that carries a user name or password is
  refused. Kirsch does not follow redirects from the provider.

## Known limits

- **A command that runs can write files.** Your approval, or the built-in
  allowlist, is the only check on what a command does. The allowlist runs
  `go build`, `go test`, `git diff`, `git log` and `ls` without asking, each
  exactly as written with nothing after it, and `go test` runs the workspace's
  own test code. A command that runs can write anywhere your account can,
  including the `.kirsch/` directory and the global configuration directory. The
  file tools refuse those paths, but a shell command is not limited that way.
- **Commands run with your privileges.** Kirsch does not sandbox them. The
  allowlist and the approval prompt are the controls.
- **Command approval can be turned off.** Setting
  `policy.require_approval_for_commands` to `false` in your global configuration
  lets every command run without asking, except shells and runners (`sh`,
  `bash`, `zsh`, `dash`, `env`, `xargs` and `nohup`), which always ask. Other
  shells and interpreters, such as `fish` or `python`, are not in that set. A
  project's configuration cannot change this setting.
- **Environment filtering works on names.** A variable whose name does not match
  one of the patterns above is passed through when it is on the passthrough list,
  even if its value is a secret. The patterns are matched case-sensitively, so a
  lowercase name such as `my_token` is not removed.
- **The debug log can hold your prompts and the paths, search queries and
  commands of tool calls.** It is off unless you turn it on with `--debug`, with
  `KIRSCH_DEBUG=1`,
  or with `telemetry.debug_log` in your global configuration. A project's
  configuration cannot turn it on. When it is on, a 400 response from the
  provider writes the full request body to the debug log. That body includes
  your prompts and any file contents sent in them. The response body in the same
  record is clipped at 256 bytes. Keep the debug log off when you work on
  material you cannot share.
- **Windows is not supported.** Process-group handling is POSIX-only.
- **Dependencies are pinned, not attested.** The module checksums in `go.sum`
  pin each dependency. Kirsch does not verify where a dependency came from, and
  its releases carry no supply-chain attestation.
