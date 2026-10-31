---
id: itd-2610070544145422
slug: someone-on-windows-installs-abcd-and-it
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Someone on Windows installs abcd and it just works: the

## Press Release

> Someone on Windows installs abcd and it just works: the tool runs in PowerShell, and the Claude Code plugin starts and runs its checks, with no WSL.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [install-update-release SOTA note](../../research/notes/2026-10-23-install-update-release-sota.md).

- **Today abcd has no Windows build**: the Makefile targets are darwin and
  linux only, the README sends Windows to WSL, and every hook is POSIX sh. The
  intent spans four surfaces: binary, installer, hooks, update verb.
- **Native Windows is a supported harness platform and Git Bash is
  optional**: without it "Claude Code uses PowerShell as the shell tool
  instead"; the harness's own installer is a PowerShell one-liner needing no
  administrator, placing the binary in `%USERPROFILE%\.local\bin` and versions
  under `%USERPROFILE%\.local\share` (the harness's setup page).
- **A shell-form hook runs in Git Bash, or in PowerShell when Git Bash is
  absent, where "a script written for Bash may not run as written"**; hooks
  take `"shell": "powershell"`, and an exec form (`command` plus `args`)
  spawns an executable directly with no shell, which on Windows must be a
  real `.exe`. A hook that fails for want of Git Bash can block the tool call
  (the harness's hooks reference). So `hooks/bootstrap.sh` does not run on a
  Windows machine without Git for Windows; the documented way out is the exec
  form calling the binary, with only the cold start left to a script.
- **Where a per-user CLI lives is contested**: Microsoft's Known Folders say
  `%LOCALAPPDATA%\Programs` for per-user programs and Go's `os.UserConfigDir`
  returns `%AppData%`; the developer-tool convention (the harness, uv, mise)
  is `~\.local\bin`, with mise keeping data in `%LOCALAPPDATA%\mise`.
- **PATH edits need a `WM_SETTINGCHANGE` broadcast** for other applications
  to see them, and a running process keeps its block (Win32 environment
  variables page); mise's PowerShell installer edits the user `Path`.
- **The running executable cannot be replaced in place**: minio/selfupdate
  renames it aside and cannot remove the `.old` file while the process runs;
  the harness notes that a WinGet upgrade can fail while it is running
  "because Windows locks the executable".
- **An unsigned binary draws a SmartScreen warning** when neither the file nor
  its signing certificate has reputation (Microsoft Defender SmartScreen
  overview); the harness's Windows binaries are Authenticode-signed; the cost
  of a managed signing service for an individual maintainer was not verified.
- **winget is the recommended Windows channel in the tools surveyed**, with an
  installer hash check and a portable installer type; the repository policy
  requires the `InstallerUrl` to be the publisher's own release location.
- **Native Windows has no host Bash sandbox, and `srt`'s Windows fence is
  alpha**: the harness's setup page marks sandboxing "Not supported" on
  native Windows (WSL 2 is the supported route), and the sandbox-runtime
  README marks Windows "Alpha". Under itd-2610161621364075's decision 7 a
  computer that cannot put up the fence stops an unattended run until the
  person acts, so what "just works" on native Windows is install, hooks and
  update; unattended drains and builds stop there by design until the fence
  does.

**Path 2 — native, shaped to the harness's own Windows conventions, with the
install channel as the seam**: `windows/amd64` and `windows/arm64` targets,
a PowerShell starter for the cold start, exec-form hooks calling the binary
(one primitive on every platform), the user-level home at
`%USERPROFILE%\.abcd.noindex` through `os.UserHomeDir`, and the asset-layout
contract shared with the starter so a later winget manifest is a second
channel rather than a second downloader. Code signing is a budget decision
with the SmartScreen warning disclosed until it is taken. No new Go
dependency. The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) Do the hooks move to the exec form on every platform, so
  one primitive serves all, or only on Windows? The POSIX bootstrap's quoting
  and `HOME` refusals were built for shell form and must be re-proved either
  way.
- (SOTA 2026-10-23) Does the starter edit the user `Path` (and broadcast the
  change), or print the line for the person to add, as `install.sh.tmpl` does
  with shell profiles? The user's directory is theirs.
- (SOTA 2026-10-23) Which proofs carry the update verb's ownership check on
  Windows, where there is no uid: the receipt and the digest alone, stated in
  the record?
- (SOTA 2026-10-23) Is code signing bought now, deferred with the SmartScreen
  warning disclosed in the README, or left to a winget channel later?
- (SOTA 2026-10-23) What does a person who already runs abcd inside WSL (the
  README's current route) get on native Windows: two installs, or a notice
  naming the other one?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
