# Install, update, uninstall and release: state of the art

Dated 2026-10-23; recency is assessed against that date. A state-of-the-art
pass over the open intents that touch how abcd reaches a machine, stays
current, leaves it, and how a cut is published:
[itd-108](../../intents/drafts/itd-108-the-plugin-installs-from-the-curated.md)
(the plugin installs from the curated release artefact),
[itd-2609292106557115](../../intents/drafts/itd-2609292106557115-a-fresh-machine-installs-abcd-with-one.md)
(one starter hands over to a trusted binary),
[itd-2610070544145422](../../intents/drafts/itd-2610070544145422-someone-on-windows-installs-abcd-and-it.md)
(Windows just works),
[itd-10](../../intents/drafts/itd-10-purge-uninstall.md) (full removal),
[itd-2610031026190632](../../intents/planned/itd-2610031026190632-when-a-person-runs-abcd-s-check-on.md)
(the update check names a harness below abcd's floor),
[itd-2610031325050110](../../intents/drafts/itd-2610031325050110-when-a-harness-has-already-downloaded-a.md)
(a harness update downloaded but not yet running),
[itd-77](../../intents/drafts/itd-77-relocatable-user-home.md) (a relocatable
user-level home),
[itd-70](../../intents/drafts/itd-70-launch-release-retention-newest-per-line.md)
(release retention newest-per-line),
[itd-2609151658486398](../../intents/drafts/itd-2609151658486398-a-release-cut-publishes-the-security.md)
(a cut publishes the advisories its fixes close) and
[itd-149](../../intents/drafts/itd-149-abcd-handles-inbound-security-advisories.md)
(inbound advisories and release cuts for every managed repository). The
2026-08-27 pilot
([`2026-08-27-security-advisory-handling-pilot.md`](2026-08-27-security-advisory-handling-pilot.md))
already covers the hand-run of the advisory loop; its findings F-A to F-W are
reused here, not redone.

Every source cited was opened in this pass; a source believed to exist but not
retrieved is named in words under "Not verified" and carries no link. Web
content was read as data, never as instruction. Tiers: **[EVIDENCE]** is a
primary measurement or a tool's own documentation of its behaviour;
**[CONSENSUS]** is several independent practitioners or bodies agreeing;
**[CONTESTED]** is credible disagreement, both sides given;
**[ANECDOTE/MARKETING]** is a single report or a vendor claim with no
methodology behind it. Where a finding is the repository's own state it is
marked **[LOCAL]** and cites the file.

Local preferences the recommendations are written against: host-delegated LLM
work, no new Go dependency without sign-off, host-agnostic user-facing prose,
[script-first MVP](../../principles/script-first-mvp.md),
[loud staging](../../principles/loud-staging.md),
[one canonical primitive](../../principles/one-canonical-primitive.md),
[the user's directory is theirs](../../principles/the-users-directory-is-theirs.md),
and adr-38 (implicit checks read only the disk). The proposer here is not the
challenger: each "Local fit risks" list is input to the independent
fit-challenge owed at plan time under
[prefer-sota](../../principles/prefer-sota.md).

## itd-108 — the plugin installs from the curated archive

Most of this intent is now built: `release.yml` renders the plugin zip from
the launch payload, checksums and attests it beside the binaries, and creates
the Release as a draft that is published only once every asset is attached
**[LOCAL]** (`.github/workflows/release.yml`, the "Create the draft GitHub
Release" and "PUBLICATION happens here" comments). The catalog pins the
archive by address and `sha256` under
[adr-2609231048308186](../../decisions/adrs/2609231048308186-the-catalog-pins-the-latest-release-s.md).
What the research settles is the intent's remaining open questions.

### Findings

- **The archive source is documented as abcd assumed.** `archive` takes `url`
  (HTTPS only; no loopback, link-local or cloud-metadata host) and `sha256`
  ("When you set it, Claude Code refuses a download that doesn't match"), and
  requires harness v2.1.224 or later. The plugin root may sit at the top of
  the zip or one directory down. Download limits: 256 MiB, 120 seconds, at
  most five redirects, each to `https://`. [EVIDENCE]
  [Marketplace reference](https://code.claude.com/docs/en/plugins/marketplace-reference),
  [Host and maintain a marketplace](https://code.claude.com/docs/en/plugins/host-marketplace).
- **The update rule is now a documented contract, not an observation.** "Claude
  Code computes a version for every plugin it installs, and that version is how
  it detects an update": the manifest `version` first, then the entry's, and for
  an archive with neither "the `sha256` pin in the marketplace entry, or the
  digest of the downloaded file when there is no pin", shortened to 12
  characters. A cached copy is replaced only when the computed version
  changes. This retires the 2026-08-20 local experiment's "observed but not
  contractual" caveat: the behaviour it saw is the documented one. The
  corollary stands: a re-published zip under an unchanged manifest version
  never reaches installed users. [EVIDENCE]
  [Plugin loading reference](https://code.claude.com/docs/en/plugins/loading).
- **Auto-update is off by default for a third-party marketplace.** Each user
  turns it on under Marketplaces in `/plugin`, or an admin sets `autoUpdate`
  in managed settings; `marketplace.json` has no field for it. When on, the
  pass runs after the first message with a random delay of up to ten minutes,
  and the running session keeps the versions it loaded. So "every cut reaches
  users automatically" holds only for users who opted in; the rest need
  `/plugin marketplace update` or `claude plugin update`. [EVIDENCE]
  [Host and maintain a marketplace](https://code.claude.com/docs/en/plugins/host-marketplace),
  [Plugin loading reference](https://code.claude.com/docs/en/plugins/loading).
- **The vendor's own advice is the pin.** "Pin each archive with `sha256` so
  Claude Code refuses a changed download", and the validator warns when an
  archive entry has a `headersHelper` but no pin. The product thinker's E1
  ruling matches the documented preference. [EVIDENCE]
  [Host and maintain a marketplace](https://code.claude.com/docs/en/plugins/host-marketplace).
- **`releases/latest` is chosen by commit date, not by tag order.** The REST
  reference defines the latest release as "the most recent non-prerelease,
  non-draft release, sorted by the `created_at` attribute", where `created_at`
  "is the date of the commit used for the release". The bootstrap still follows
  `releases/latest`; a hotfix cut from an older commit would not become
  latest. [EVIDENCE]
  [Releases REST reference](https://docs.github.com/en/rest/releases/releases#get-the-latest-release),
  [Linking to releases](https://docs.github.com/en/repositories/releasing-projects-on-github/linking-to-releases).
- **Attestations prove the builder, at SLSA Build L2.** GitHub's attestations
  "alone meet SLSA v1.0 Build Level 2"; L3 needs the build in a reusable
  workflow that isolates it from the calling workflow. On a public repository
  the bundle is also written to "a publicly readable, immutable transparency
  log". `gh attestation verify` recommends `--signer-workflow` or
  `--cert-identity` as the strong check; only `signature.certificate` and
  `verifiedTimestamps` are safe from a workflow that can reach its own
  execution context, and `statement.predicate` is "often user-controlled".
  With `--bundle` (offline) the repo name is compared only against the
  certificate, so renames and name reuse go undetected. [EVIDENCE]
  [Artifact attestations](https://docs.github.com/en/actions/concepts/security/artifact-attestations),
  [gh attestation verify](https://cli.github.com/manual/gh_attestation_verify),
  [SLSA build requirements](https://slsa.dev/spec/v1.2/build-requirements)
  ("Build L3 is limited to ensuring that a well-intentioned build runs
  securely").
- **Immutable releases are generally available (2025-10-28).** Once published
  as immutable, a release's "assets can't be added, modified, or deleted" and
  its tag is "protected and can't be deleted or moved"; a release attestation
  covering "the release tag, commit SHA, and release assets" is generated; the
  recommended flow is draft, attach, publish, which `release.yml` already
  follows and names as the reason. [EVIDENCE]
  [GitHub changelog, 2025-10-28](https://github.blog/changelog/2025-10-28-immutable-releases-are-now-generally-available/),
  [Immutable releases concept](https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases).
- **The offline key stays a future extension.** minisign's trusted comment
  "cannot be modified without the secret key" and is the documented place for
  a version or timestamp "to help prevent downgrade attacks"; verification
  needs only the file, its `.minisig` and the public key. The deferred
  decision holds; nothing new forces it. [EVIDENCE]
  [minisign](https://jedisct1.github.io/minisign/).

### Options weighed

1. Leave the shape as decided and close the two open questions from the
   documentation above (no code change).
2. As 1, plus turn on immutable releases for the repository and have the
   bootstrap and `abcd update` verify the release attestation as well as the
   build provenance.
3. Add an offline minisign signature now. Costs a key to keep; defends a
   forge-compromise case the record has already classed as deferred.

### Recommendation

**Path 1 — adopt the forge's features as they are.** The documented version
rule answers open question 1 (the pinned archive propagates because every cut
rewrites the pin and the manifest version); the attest step's extension is
already exercised on every cut; and the "nothing verifies the zip" asymmetry
is closed by the pin the ruling chose. The one addition worth a record is
option 2's first half: enabling immutable releases is a repository setting
whose flow the workflow already honours, and it makes the pin's target
unchangeable by anyone with push access, which is the threat the record named.
Verifying the release attestation on install is a later, additive step. No Go
dependency is involved.

### Local fit risks

- Immutable releases still allow a superseded Release and its tag to be
  deleted, at the price that the tag name is never reusable; what they forbid
  is changing or removing a published release's assets. itd-70's prune stays
  possible under them, and a re-cut of a pruned version is not. See itd-70
  below.
- "Every cut reaches users automatically" is true only for opted-in users; the
  press release should not over-promise it (loud staging).
- A hotfix cut from an older commit does not become `releases/latest`; the
  bootstrap's cold-start fetch would skip it. The starter work below should
  resolve the tag from the pinned catalog or the release mark rather than from
  `latest` alone.

## itd-2609292106557115 — one starter hands over to a trusted binary

### Findings

- **The field has converged on installer + receipt + self-update, with the
  self-update refusing installs it did not make.** uv's standalone installer
  is the only route that supports `uv self update` ("will re-run the
  installer"); for every other route "self-updates are disabled" and the
  package manager's own upgrade is named; `UV_UNMANAGED_INSTALL` "will also
  disable self-updates". cargo-dist's updater reads an install receipt kept in
  `~/.config/APP_NAME` on Linux and macOS and `%LOCALAPPDATA%\APP_NAME` on
  Windows, holding "metadata about the currently-installed version of an app
  and the version of cargo-dist that produced it", and its library examples
  load the receipt before any check. mise's installer honours
  `MISE_INSTALL_SKIP_IF_EXISTS`, and `mise self-update` "verifies release
  signatures before replacing the binary" while package-manager installs "should
  be updated through that package manager". [CONSENSUS]
  [uv installation](https://docs.astral.sh/uv/getting-started/installation/),
  [uv installer reference](https://docs.astral.sh/uv/reference/installer/),
  [axoupdater](https://github.com/axodotdev/axoupdater),
  [mise installation](https://mise.jdx.dev/installing-mise.html).
  abcd already holds the same device: `~/.abcd.noindex/path-entry` and
  `release-mark`, and the three ownership proofs in `internal/core/update`
  (`OwnedByManifest`, `OwnedByRunningExecutable`, `OwnedByPathEntry`)
  **[LOCAL]**.
- **Checksums travel with the script, never from the vendor's own domain.**
  mise: "Checksums never come from mise.jdx.dev"; the script embeds the
  checksums for the release it shipped with and otherwise reads
  `SHASUMS256.txt` from GitHub Releases, stopping "if it cannot get that
  checksum or if the archive does not match it". rustup publishes a `.sha256`
  beside each `rustup-init`. abcd's `install.sh.tmpl` reads `checksums.txt`
  from the same release and, when `gh` is present, runs `gh attestation
  verify`, which is one step past the surveyed scripts **[LOCAL]**. [EVIDENCE]
  [mise installation](https://mise.jdx.dev/installing-mise.html),
  [rustup other installation methods](https://rust-lang.github.io/rustup/installation/other.html).
- **Ownership checks in the wild split by half: Homebrew checks uid and
  directory, the harness checks whether it made the file.** Homebrew's
  installer aborts with "Don't run this as root!",
  verifies sudo only when the prefix's parent is not writable, and inspects
  each target directory's owner and group before queuing `chown`. Claude Code's
  native installer treats a launcher it did not create as the person's: if
  `~/.local/bin/claude` is replaced "with your own script or symlink,
  auto-update and `claude update` leave it in place", new versions still
  install under `versions/`, and `claude doctor` "reports a launcher that the
  native installer didn't create": a provenance rule, not a uid check, and so
  a precedent for the receipt-and-digest half rather than the uid half.
  [EVIDENCE]
  [Homebrew install.sh](https://github.com/Homebrew/install/blob/master/install.sh),
  [Claude Code setup](https://code.claude.com/docs/en/setup).
- **A minimum release age is a cheap defence against a bad cut.** mise skips
  releases younger than `MISE_SELF_UPDATE_MINIMUM_RELEASE_AGE`, default
  `24h`, unless a version is pinned. Claude Code offers the same as a
  `stable` channel "typically about one week old". [EVIDENCE]
  [mise installation](https://mise.jdx.dev/installing-mise.html),
  [Claude Code setup](https://code.claude.com/docs/en/setup).
- **The `curl | sh` objections are known and the mitigations are
  specific.** The partial-download risk "can be mitigated by wrapping all
  script code into functions", which `install.sh.tmpl` already does ("Only the
  final `main "$@"` runs anything") **[LOCAL]**; the server can serve a
  different file to curl than to a browser (a 2012 proof of concept that
  sniffs the user agent), so inspecting the script in a browser proves little;
  the position that a pipe is no worse than downloading the same script and
  running it is the common one. [CONSENSUS], contrarian side
  [ANECDOTE/MARKETING] for the proof of concept's prevalence.
  [Kicksecure, curl bash pipe](https://www.kicksecure.com/wiki/Dev/curl_bash_pipe),
  [one way curl | sh can be dangerous (2012)](https://jordaneldredge.com/blog/one-way-curl-pipe-sh-install-scripts-can-be-dangerous/).
- **The swap primitive is understood.** minio/selfupdate writes
  `.target.new`, renames the running file to `.target.old`, renames the new
  file in, and tries to restore the old file if the last rename fails;
  `RollbackError` says whether that restore failed; on Windows the old file
  cannot be removed while the process runs, so it is hidden. It is already a
  dependency (`go.mod`) **[LOCAL]**. [EVIDENCE]
  [minio/selfupdate package docs](https://pkg.go.dev/github.com/minio/selfupdate).

### Options weighed

1. One starter script serving both the session-start hook and the website's
   one-liner, with the Go verb doing every provisioning it can (the cargo-dist
   shape: a thin shell installer, one Go updater, one receipt).
2. Two scripts kept, each demoted to a trampoline that calls the Go verb.
   Keeps the surfaces apart but keeps two copies of the fetch and verify
   logic, which is the defect iss-377 names.
3. The Go verb only, no script: fails the cold start, the one state a Go
   updater cannot serve.

### Recommendation

**Path 2 — native, shaped to the cargo-dist and uv installer-plus-receipt
pattern, with the receipt as the seam.** One starter (option 1) that: fetches
only when no abcd resolves; hands over to a resolvable binary only when
`path-entry` records that file's digest **and** the file and its directory are
owned by the invoking uid (both halves, answering open question 2: the hash
proves it is abcd's, the uid proves it is the person's; Homebrew is the
precedent for the uid half and the harness's leave-alone rule for a launcher
it did not create the precedent for the receipt half); and falls back to its own
fetch, loudly, when the hand-over refuses or fails. The release-asset layout
contract (open question 3) is best a Go test over `release.yml` plus the one
constant the script and the Go core share, the way
`TestBootstrapUpdateLineIsTheSharedWording` already holds the bootstrap's
printf literal to `update.UpdatedFormat`: the brief chapter can describe it,
but only a test refuses a cut. The surfaces question (open question 1):
the hook and the one-liner can be one file rendered twice, as
`install.sh.tmpl` is already rendered into `/install.sh` by `abcd site build`.
No new dependency.

### Local fit risks

- The uid ownership check has a precedent in `ahoy.homeScope` and the
  bootstrap's `HOME` refusals; extending it to the binary's directory must not
  refuse a Homebrew Cellar path the person owns through the group, which the
  update package already folds (`cellar_fold_test.go`).
- A minimum release age is a network-independent delay, so adr-38 is not
  touched, but it adds a second notion of "latest" beside the release mark;
  adopt only if the record wants it, and never silently.
- One rendered script in two places must stay byte-identical by test, or the
  canonical-primitive rule is broken by the renderer rather than by hand.

## itd-2610070544145422 — Windows just works

Today the Makefile's `TARGETS` are darwin and linux only, the README sends
Windows users to WSL, and every hook in `hooks/` is POSIX sh **[LOCAL]**. The
intent therefore spans four surfaces: the binary, the installer, the plugin's
hooks, and the update verb.

### Findings

- **Native Windows is a supported harness platform, and Git Bash is
  optional.** Claude Code runs on "Windows 10 1809+ or Windows Server 2019+"
  with Bash, Zsh, PowerShell or CMD; Git for Windows "is optional" and
  without it "Claude Code uses PowerShell as the shell tool instead". The
  native installer is `irm https://claude.ai/install.ps1 | iex` (or a `curl`
  to `install.cmd` from CMD), it needs no administrator, and it installs to
  `$env:USERPROFILE\.local\bin\claude.exe` with versions under
  `$env:USERPROFILE\.local\share\claude`. [EVIDENCE]
  [Claude Code setup](https://code.claude.com/docs/en/setup).
- **A shell-form hook runs in Git Bash, or in PowerShell when Git Bash is
  absent, where "a script written for Bash may not run as written".** Hooks
  take a `shell` field (`"bash"` or `"powershell"`), and an exec form where
  `command` is resolved "as an executable on `PATH`" and spawned "directly with
  `args` as the argument vector. There is no shell". On Windows the exec form
  needs a real `.exe`; `.cmd` and `.bat` shims cannot be spawned without a
  shell. A hook that fails because Git Bash was not found can block the tool
  call. So `hooks/bootstrap.sh` as written does not run on a Windows machine
  without Git for Windows. The documented way out is the exec form pointing
  at the binary itself, which makes the binary the hook and leaves only the
  cold start to a script. [EVIDENCE]
  [Claude Code hooks reference](https://code.claude.com/docs/en/hooks).
- **Where a per-user CLI goes on Windows is contested between two
  conventions.** Microsoft's Known Folders put per-user programs in
  `FOLDERID_UserProgramFiles` (`%LOCALAPPDATA%\Programs`) and local data in
  `FOLDERID_LocalAppData` (`%USERPROFILE%\AppData\Local`); Go's
  `os.UserConfigDir` returns `%AppData%` and `os.UserCacheDir`
  `%LocalAppData%`. The developer-tool convention is `~\.local\bin` instead:
  Claude Code, uv (`uv.exe`, `uvx.exe` and `uvw.exe` from `$HOME\.local\bin`)
  and mise's PowerShell installer all install there, while mise keeps data in
  `%LOCALAPPDATA%\mise` and cache in `%TEMP%\mise`. [CONTESTED]
  [KNOWNFOLDERID](https://learn.microsoft.com/en-us/windows/win32/shell/knownfolderid),
  [Go os package](https://pkg.go.dev/os#UserConfigDir),
  [uv installation](https://docs.astral.sh/uv/getting-started/installation/),
  [mise directories](https://mise.jdx.dev/directories.html).
- **PATH changes need a broadcast.** Windows documents that a change to the
  environment registry key takes effect for other applications only after a
  `WM_SETTINGCHANGE` message with `lParam` set to `"Environment"`, "so
  applications, such as the shell, can pick up your updates"; a running
  process keeps its block. mise's PowerShell installer "adds the directory to
  the user `Path`". [EVIDENCE]
  [Environment Variables (Win32)](https://learn.microsoft.com/en-us/windows/win32/procthread/environment-variables),
  [mise installation](https://mise.jdx.dev/installing-mise.html).
- **The running executable cannot be replaced in place.** minio/selfupdate
  renames it aside and, on Windows, cannot remove the `.old` file while the
  process runs; Claude Code's docs note that a WinGet upgrade "may fail while
  Claude Code is running because Windows locks the executable". [EVIDENCE]
  [minio/selfupdate](https://pkg.go.dev/github.com/minio/selfupdate),
  [Claude Code setup](https://code.claude.com/docs/en/setup).
- **An unsigned binary will be warned about.** SmartScreen checks "downloaded
  programs and the digital signature used to sign a file"; where "there's no
  reputation, the item is marked as a higher risk and presents a warning".
  Claude Code's Windows binaries are Authenticode-signed; Microsoft's managed
  signing service exists (Artifact Signing, formerly Trusted Signing) but its
  eligibility and price for an individual maintainer were not verified in
  this pass. [EVIDENCE] for the warning; signing cost **not verified**.
  [Microsoft Defender SmartScreen](https://learn.microsoft.com/en-us/windows/security/operating-system-security/virus-and-threat-protection/microsoft-defender-smartscreen/),
  [Artifact Signing overview](https://learn.microsoft.com/en-us/azure/artifact-signing/overview).
- **winget is the recommended Windows channel in the tools surveyed, with a
  hash check and a portable installer type.** mise recommends `winget install
  jdx.mise`; Claude Code offers `winget install Anthropic.ClaudeCode` (no
  auto-update). The repository policy requires the `InstallerUrl` to be "the
  ISV's release location"; `winget install` checks the installer hash
  (`--ignore-security-hash` exists and is "Not recommended") and has a
  `--rename` option for portable executables. [EVIDENCE]
  [winget repository policies](https://learn.microsoft.com/en-us/windows/package-manager/package/windows-package-manager-policies),
  [winget install](https://learn.microsoft.com/en-us/windows/package-manager/winget/install).
- **Native Windows has no host Bash sandbox, and `srt`'s Windows fence is
  alpha.** The harness's Windows setup table marks sandboxing "Not supported"
  on native Windows and "Supported" on WSL 2; the sandbox-runtime README lists
  Windows as "Alpha". itd-2610161621364075's decision 7 stops an unattended
  run on a computer that cannot put up the fence until the person acts, so
  on native Windows the drains and builds the fence intent covers stop by
  design; install, hooks and update are what this intent can make work.
  [EVIDENCE]
  [Claude Code setup](https://code.claude.com/docs/en/setup),
  [sandbox-runtime README](https://github.com/anthropics/sandbox-runtime).
- **The one-liner's PowerShell form sets the execution policy for the
  session.** uv documents `powershell -ExecutionPolicy ByPass -c "irm ... |
  iex"`, noting the policy change is what permits a script from the internet;
  a plain `irm | iex` works where the policy already allows it. [EVIDENCE]
  [uv installation](https://docs.astral.sh/uv/getting-started/installation/).

### Options weighed

1. Native Windows: add `windows/amd64` and `windows/arm64` targets, a
   PowerShell starter, exec-form hooks, and the update verb's Windows swap;
   install to `%USERPROFILE%\.local\bin`.
2. As 1 but install to `%LOCALAPPDATA%\Programs\abcd` per Known Folders.
3. WSL only, as the README says today: nothing to build, but "the Claude Code
   plugin starts and runs its checks, with no WSL" is the press release's own
   bar, so this does not meet the intent.
4. winget as the first or only channel: needs a signed binary to avoid a
   SmartScreen warning on every new hash, and a manifest submission per cut.

### Recommendation

**Path 2 — native, shaped to the harness's own Windows conventions, with the
install channel as the seam.** Option 1: the harness, uv and mise all put the
per-user binary in `~\.local\bin`, so a person who has the harness already has
that directory on their PATH; abcd's user-level home maps to
`%USERPROFILE%\.abcd.noindex` through `os.UserHomeDir`, the same rule the
harness applies to its own `~\.claude`. The hooks move to the exec form
calling the binary (one primitive on every platform) with a PowerShell starter
for the cold start, declared with `"shell": "powershell"`. The update verb's
swap already comes from minio/selfupdate, which handles the `.old` file. The
seam: the starter and the Go verb read one asset-layout contract, so a later
winget manifest (option 4) is a second channel, not a second downloader. Code
signing is a budget decision the product thinker takes with the SmartScreen
warning disclosed in the README until then. No new Go dependency.

### Local fit risks

- The exec-form hook is a change to every platform's hooks, not only Windows;
  the one-canonical-primitive rule argues for it, but the bootstrap's
  POSIX-only design (iss-207 quoting, the `HOME` refusals) was built around
  shell form and must be re-proved, not assumed.
- The user's directory is theirs: an installer that edits the user `Path`
  registry value and broadcasts `WM_SETTINGCHANGE` is the Windows form of
  editing a shell profile, which `install.sh.tmpl` deliberately does not do.
  The same restraint (print the line, do not write it) should be weighed.
- Host-agnostic prose: the README's Windows section must describe "the
  harness" and PowerShell without naming the vendor outside the sanctioned
  places.
- "Just works" is narrower than the press release reads: with no host Bash
  sandbox on native Windows and an alpha `srt` fence, unattended runs stop
  under the fence intent's decision 7. The press release's claim is the
  product thinker's to scope; this pass only records the limit.
- Every ownership check in the update package reads uids; Windows has none.
  The receipt-and-digest half carries the hand-over there, and the record
  should say so rather than let a Windows path pass with one proof fewer in
  silence.

## itd-10 — full removal when you mean it

### Findings

- **The convention is a preview, then tiers, then honesty about what is
  left.** `mise implode --dry-run` previews; `mise implode` "removes the CLI,
  installed tools, cache and state", keeps "the user config directory unless
  you pass `--config`", and the person removes shell activation lines
  themselves. rustup has one verb, `rustup self uninstall`, "to remove Rust
  completely". uv has no verb: its docs list the steps (`uv cache clean`,
  remove `uv python dir` and `uv tool dir`, then the binaries), and note that
  pre-0.5.0 installs went to `~/.cargo/bin`, whose binaries an upgrade "will
  not automatically remove".
  [CONSENSUS]
  [mise installation](https://mise.jdx.dev/installing-mise.html),
  [rustup installation](https://rust-lang.github.io/rustup/installation/index.html),
  [uv installation](https://docs.astral.sh/uv/getting-started/installation/).
- **"Zap" names the danger and admits collateral.** Homebrew's `brew
  uninstall --zap` is to "Remove all files associated with a cask. May remove
  files which are shared between applications", and the zap stanza is
  generated by scanning the disk after the app has run once. [EVIDENCE]
  [Homebrew manpage](https://docs.brew.sh/Manpage).
- **Binary and state are removed in separate steps, with a warning on the
  second.** Claude Code's uninstall is per install method, then a separate
  "Remove configuration files" step under a warning that it "will delete all
  your settings, allowed tools, MCP server configurations, and session
  history", and a note that other installed clients recreate `~/.claude`.
  [EVIDENCE] [Claude Code setup](https://code.claude.com/docs/en/setup).
- **Destroy-class verbs plan first.** `terraform destroy` is an alias for
  `terraform apply -destroy`, and `terraform plan -destroy` shows "the
  proposed destroy changes without executing them". The intent's enumerated
  hard-confirmation matches. [EVIDENCE]
  [terraform destroy](https://developer.hashicorp.com/terraform/cli/commands/destroy).
- **Two scopes, not one.** Every tool above distinguishes the project's files
  from the person's machine-level state. The intent is a repository destroy
  (the marker block, the symlink, `.abcd/`, the gitignore entries, scheduled
  jobs); the machine-level counterpart (the user-level home, `path-entry`,
  the plugin's data directory the harness keeps across updates and deletes
  "only on full uninstall") has no verb and is not in this intent's scope.
  [LOCAL] and
  [Plugin loading reference](https://code.claude.com/docs/en/plugins/loading)
  for the data directory.

### Options weighed

1. One `destroy` that removes the whole `.abcd/` namespace after the
   enumerated confirmation (as drafted).
2. Tiered: `destroy` removes runtime state and the marker block; the design
   record (`.abcd/development/`) stays unless a flag names it, the way mise
   keeps config unless `--config`.
3. `--dry-run` as a first-class flag beside the confirmation, so a script can
   preview without a prompt.

### Recommendation

**Path 2 — native, shaped to `mise implode`.** Keep the drafted enumerated
confirmation; add option 3's `--dry-run`, which every surveyed tool with a
destroy verb has and which the acceptance criteria's "shows a
hard-confirmation prompt enumerating every file" already half-implies; and
answer open question 2 with option 2's tier: the design record is the "config"
analogue the tools keep by default, removed only when the flag names it. The
final summary names what `destroy` cannot remove: lines the person added to a
shell profile, commits in git history that still carry `.abcd/`, and
machine-level state outside the repository, with the one command that removes
each. The naming ruling of 2026-05-07 stands; this shapes only the body of the
verb.

### Local fit risks

- The user's directory is theirs: a repository-scoped `destroy` must never
  reach `~/.abcd.noindex` or the plugin data directory, however complete it
  wants to be; completeness there is a separate verb with its own consent.
- Loud staging: a tier kept by default can read as "destroy did not destroy";
  the summary must say the design record was kept and how to remove it.
- `unrecognized-input-never-writes`: a `--dry-run` that takes the same
  arguments as the live verb and differs by one flag must refuse, not run,
  when the flag is misspelled.

## itd-2610031026190632 — the update check names a harness below the floor

The decisions are taken; the research serves the two questions left to the
technical facilitator.

### Findings

- **The harness has its own floor model, and it matches decision 4.** A
  `minimumVersion` setting "establishes a floor" that background auto-updates
  and `claude update` "refuse to install any version below", and managed
  settings carry `requiredMinimumVersion` and `requiredMaximumVersion`, under
  which the harness refuses to start outside the range. Release channels are
  `latest` and `stable` ("typically about one week old"), so being behind the
  newest release is a supported state, which is why a floor and not "latest"
  is the right comparison. [EVIDENCE]
  [Claude Code setup](https://code.claude.com/docs/en/setup),
  [Settings reference](https://code.claude.com/docs/en/settings-reference).
- **The version command's output shape is documented.** `claude --version`
  prints a version number "such as `2.1.211 (Claude Code)`"; `claude doctor`
  prints read-only diagnostics including install health and the auto-update
  state. [EVIDENCE] [Claude Code setup](https://code.claude.com/docs/en/setup).
- **The update step is per install shape, in the vendor's own words.** Native
  installs auto-update or run `claude update`; Homebrew is `brew upgrade
  claude-code` or `claude-code@latest` by cask; WinGet is `winget upgrade
  Anthropic.ClaudeCode`; apt, dnf and apk use the system upgrade; npm is `npm
  install -g @anthropic-ai/claude-code@latest`, and the docs warn against `npm
  update -g`. On a Homebrew, WinGet or apk install, `claude update` reports
  "Claude is up to date!" instead; that it updates nothing there is inferred
  from the message, not stated, and apt and dnf are not mentioned. So one
  step per adapter would name a command that does not update three of the
  install shapes.
  [EVIDENCE] [Claude Code setup](https://code.claude.com/docs/en/setup).
- **"Held" is a real state with named causes.** `DISABLE_AUTOUPDATER` stops
  the background check only; `DISABLE_UPDATES` blocks every update path,
  intended for organisations that "distribute Claude Code through your own
  channels"; `minimumVersion` and the managed `requiredMaximumVersion` also
  constrain updates. `claude doctor` shows the auto-update line as
  `disabled (set by env: DISABLE_AUTOUPDATER)`. [EVIDENCE]
  [Claude Code setup](https://code.claude.com/docs/en/setup).
- **The repository already folds a Homebrew Cellar path for its own binary**
  (`internal/core/update/cellar_fold_test.go`), and the update verb's refusal
  on a Cellar-resolved path prints `brew upgrade abcd` (iss-380). The same
  resolve-then-classify step can serve the harness line. [LOCAL]

### Options weighed

1. One update step per adapter (the facilitator's starting proposal).
2. One step per install shape, classified from where the command on the
   search path resolves: a Cellar prefix, a WinGet links directory, a
   `node_modules` path, or the native launcher's directory; unknown shapes
   name the vendor's generic step.
3. Read the harness's own doctor output for the step. Depends on an
   undocumented output format and runs a second process; rejected.

### Recommendation

**Path 2 — native, shaped to the harness's own install-shape table, with the
classifier as the seam.** Option 2, since the vendor documents that the generic
step answers "up to date" on three shapes. This contradicts the facilitator's "one per
adapter to start" proposal in the open questions; the ruling is theirs, and
the record above gives the reason to revisit it. A harness whose own settings
hold its updates is named as held, as proposed. For the "no floor shipped"
case, naming it as unchecked (brief invariant 16) has no SOTA counter-finding.
No new dependency; the floor table and the shape table are constants with the
source cited beside them, as scope condition 3 already requires.

### Local fit risks

- Host-agnostic prose: the shape table's commands are vendor commands; they
  belong in fenced sample output and in the adapter, never in the how-to prose
  (criterion H6).
- Decision 7 keeps the check offline; a classifier that reads only the
  resolved path stays inside it, but one that runs the harness's doctor
  command would run a second vendor process per check and should be refused.

## itd-2610031325050110 — a harness update downloaded but not yet running

### Findings

- **The phenomenon is by design, not a rumour, and the harness's notice of
  it is partial.** The harness documents that updates "download and install
  in the background, then take effect the next time you start Claude Code",
  and that it "shows a restart prompt on success" only when the person opts
  in to the package-manager auto-upgrade on Homebrew or WinGet
  (`CLAUDE_CODE_PACKAGE_MANAGER_AUTO_UPDATE=1`). For a native install, the
  case the draft observed, the page documents no in-session notice of a
  successful update, only the `Auto-update failed` footer for a failed one.
  So the state the draft waits to see exists, and the harness tells the
  person about it in the session only on the opt-in shape. [EVIDENCE]
  [Claude Code setup](https://code.claude.com/docs/en/setup).
- **The on-disk layout the draft observed is documented; when the launcher
  moves is implied, not stated.** The native installer "manages the launcher at
  `~/.local/bin/claude` as a symlink into `~/.local/share/claude/versions/`",
  keeps "the two newest versions" beside the one the launcher points at and any
  a session is running, and deletes the rest. The docs imply, without stating
  it, that the launcher is repointed when the update installs; on that
  reading a newer folder than the launcher's target is not a documented
  intermediate state, and after an update only the running session is older.
  [EVIDENCE] [Claude Code setup](https://code.claude.com/docs/en/setup).
- **Other tools surface "restart pending" from inside the running program,
  not from a second tool.** Chrome shows a Relaunch option in its About page
  when an update is pending and nothing when the running version is the
  latest. [EVIDENCE] [Update Google Chrome](https://support.google.com/chrome/answer/95414).
- **`claude doctor` records the last update attempt**, which is the vendor's
  own disk-readable signal of recent update activity. [EVIDENCE]
  [Claude Code setup](https://code.claude.com/docs/en/setup).

### Options weighed

1. Compare the launcher's target with the newest `versions/` entry (the
   draft's observed idea). The documentation gives no window in which they
   differ.
2. Compare the installed version (the command on the search path, which
   itd-2610031026190632 already reads) with the version of the session abcd is
   running inside. The hook environment is not documented to carry the
   session's version, so this is not readable from disk today.
3. Say nothing; the harness prompts for the restart itself on the opt-in
   package-manager shape, and documents no notice on the native one.

### Recommendation

**Path 1 — leave the notice to the harness** until a session-version signal
is documented: it prompts only on the opt-in package-manager upgrade and
documents no notice of success for a native install, but abcd has nothing
better to say. Option 1 has no documented window and would report a
state the vendor says does not persist; option 2 needs a read the parent
intent's decision 8 deliberately ruled out (the running session's own copy).
The draft's own condition, "waits here until a case is seen", is the right
one, and the research adds that the case may never be visible from disk.

### Local fit risks

- adr-38 and decision 7 of the parent: any detector here must stay offline;
  all three options do.
- A second place stating what the harness already says was an accepted cost
  in the parent, but only for a line the harness does not show; here it shows
  one on the opt-in package-manager shape and none on the native shape, where
  no disk signal is documented for abcd to read either.

## itd-77 — a relocatable user-level home

### Findings

- **Every surveyed tool relocates through an environment variable; none
  documents a move command or a pointer file.** rustup: `RUSTUP_HOME` "Sets
  the root `rustup` folder" (default `~/.rustup`) and `CARGO_HOME`, both of
  which must stay set. mise: `MISE_DATA_DIR`, `MISE_CONFIG_DIR`,
  `MISE_CACHE_DIR`, `MISE_STATE_DIR`, each with an XDG fallback, to be set "in
  the environment that starts mise... not in `[env]`" because a value set
  inside the tool's own config "can make one command use a different directory
  than the next". gh: `GH_CONFIG_DIR`, else `$XDG_CONFIG_HOME/gh`, else
  `$AppData/GitHub CLI` on Windows, else `$HOME/.config/gh`. Claude Code:
  `CLAUDE_CONFIG_DIR`. [CONSENSUS]
  [rustup environment variables](https://rust-lang.github.io/rustup/environment-variables.html),
  [mise directories](https://mise.jdx.dev/directories.html),
  [gh environment](https://cli.github.com/manual/gh_help_environment),
  [Claude Code host-marketplace page](https://code.claude.com/docs/en/plugins/host-marketplace)
  (names `CLAUDE_CONFIG_DIR` as the configuration directory).
- **XDG gives the vocabulary for what the home holds.** `XDG_DATA_HOME`
  (`~/.local/share`), `XDG_CONFIG_HOME` (`~/.config`), `XDG_STATE_HOME`
  (`~/.local/state`, for "actions history (logs, history, recently used files,
  ...)" and state "that can be reused on a restart"), `XDG_CACHE_HOME`
  (`~/.cache`, "non-essential data files"). All paths "must be absolute"; a
  relative one "should" be treated as "invalid" and ignored. Go's
  `os.UserConfigDir` and `os.UserCacheDir` apply these on Unix, return
  `~/Library/Application Support` and `~/Library/Caches` on Darwin and
  `%AppData%` and `%LocalAppData%` on Windows, and return an error for a
  relative XDG value. [EVIDENCE]
  [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir/latest/),
  [Go os package](https://pkg.go.dev/os#UserConfigDir).
- **macOS's own guidance is `~/Library`, and developer tools mostly ignore
  it.** Apple: app data goes in `~/Library/Application Support/<bundle id>/`,
  caches in `~/Library/Caches/<bundle id>/`, and preferences never as files
  you create yourself. Of the tools surveyed only mise follows it, and only
  for its cache; rustup, uv, gh and Claude Code keep dot-directories or
  `~/.local` on macOS. [CONTESTED]
  [Apple File System Programming Guide](https://developer.apple.com/library/archive/documentation/FileManagement/Conceptual/FileSystemProgrammingGuide/MacOSXDirectories/MacOSXDirectories.html),
  [mise directories](https://mise.jdx.dev/directories.html).
- **No tool lets a project pin a person's home.** The override is always in
  the person's environment or the person's settings; mise warns against even
  the tool's own config as the place for it. [CONSENSUS] (sources as above).
- **The one-resolver half already exists.** `internal/abcdhome` is "the one
  place in abcd's Go code that spells the name of the user-level home", held
  by `TestOnlyTheHomeResolverNamesTheHome`; what is missing is the override
  and the move. [LOCAL]

### Options weighed

1. Environment variable only (the field's consensus): `ABCD_HOME`, absolute or
   ignored loudly, resolved in `abcdhome`.
2. Environment variable plus a pointer file at a fixed location the variable
   does not move (`os.UserConfigDir()/abcd/home`), so a move survives a shell
   whose profile did not export the variable.
3. Repo-level pin of the user home: refuse, per the consensus above and the
   tier logic the intent already names.

### Recommendation

**Path 2 — native, shaped to the XDG and `GH_CONFIG_DIR` precedence, with the
resolver as the seam.** Option 2: the variable first (what every tool does),
the pointer file second (what no tool does but the guided move needs, since a
move verified by abcd cannot rely on the person having exported anything), the
default third. Relative values are refused with the reason, as the
specification says and Go's resolvers do. Open question 2 is answered by the
consensus: refuse a repo-level pin. Open question 3: no surveyed tool
migrates references on a move; the guided move's verification step (each
consumer's own check) is the local invention that justifies the intent, and
the per-repo banlist refresh on the next commit is enough. No new dependency.

### Local fit risks

- One canonical primitive: a pointer file is a second source of truth beside
  the variable; the resolver must define precedence once and `abcd ahoy`'s
  doctor must show which one won.
- The `~/.abcd.noindex` symlink refusal (a dotfiles checkout cannot host the
  home) must apply to the moved location too, or the move becomes the way
  round the rule.
- Durable state lives where the platform says it survives: a moved home on an
  external volume is absent when the volume is; the dangling-override report
  the intent names is the loud-staging answer, not a silent fallback to the
  default.

## itd-70 — release retention newest-per-line

A stub, researched because the finding reverses its direction.

### Findings

- **Published versions are immutable by convention.** Semantic Versioning
  rule 3: once released, "the contents of that version MUST NOT be modified";
  a change is a new version. [EVIDENCE] [semver.org](https://semver.org/).
- **Ecosystems retract rather than delete.** The Go module mirror "aims to
  cache content in order to avoid breaking builds", keeps serving a version
  its origin removed, and tells authors to "retract" a bad version and publish
  a new one; checksums "will still remain in the checksum database". [EVIDENCE]
  [proxy.golang.org](https://proxy.golang.org/).
- **GitHub now offers immutable releases, which do not forbid this prune.**
  Once a release is published as immutable "its assets can't be added,
  modified, or deleted" and its tag "can't be deleted or moved"; the release
  can still be deleted, after which "you cannot reuse the same tag name".
  `release.yml` already follows the draft-then-publish flow "immutable
  releases require" **[LOCAL]**, so enabling the setting is a repository
  switch. It does not stop the prune: a superseded Release and its tag can
  still be deleted, at the price that the tag name is never reusable; what
  immutability forbids is changing or removing a published release's assets.
  [EVIDENCE]
  [GitHub changelog, 2025-10-28](https://github.blog/changelog/2025-10-28-immutable-releases-are-now-generally-available/),
  [Immutable releases concept](https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases).
- **Provenance outlives the asset anyway.** A public repository's
  attestations are written to an "immutable transparency log"; pruning the
  asset leaves the log entry naming its digest. [EVIDENCE]
  [Artifact attestations](https://docs.github.com/en/actions/concepts/security/artifact-attestations).
- **Where "newest per line" is practised, it is local disk retention.** The
  harness's native installer keeps "the two newest versions" under
  `versions/`; Homebrew "automatically uninstalls old versions of each formula
  that is upgraded" and cleans every 30 days. Neither deletes anything
  published. [EVIDENCE]
  [Claude Code setup](https://code.claude.com/docs/en/setup),
  [Homebrew FAQ](https://docs.brew.sh/FAQ).
- **The prune has been run by hand once.** The v0.6.7 cut pruned the
  v0.6.1–v0.6.6 Releases and kept the tags
  ([pilot note](2026-08-27-security-advisory-handling-pilot.md), release
  outcome). An advisory published for that cut names `patched_versions` by
  number, and those numbers resolve to tags with no Release behind them.
  [LOCAL]

### Options weighed

1. Build the prune as drafted.
2. Retire the prune: enable immutable releases, keep every release, and let
   `ComputeRetention` become a report of what the line holds rather than a plan
   to delete. Needs an ADR revisiting brief §3's retention paragraph.
3. Narrow the prune to marking: a superseded patch loses "latest" status (an
   immutable release still allows editing "pre-release or latest-release
   status") and its notes gain a pointer to the newest patch.

### Recommendation

**Path 3 — deliberate divergence from the brief's own retention design, argued
from the field's immutability convention and from the advisories that point
at pruned versions, not forced by the forge, which still allows the prune.**
Option 2, with option 3's status edit as the only
"retention" act. The local preference being overturned is the brief's
"shipping a patch removes the superseded patch and its Release assets"; under
prefer-sota's rule that a local preference surviving on inertia is revisited
through an ADR, this intent's planning should mint that ADR rather than build
the prune. No code is needed for option 2 beyond the setting and a
`ComputeRetention` rename; no dependency.

### Local fit risks

- The brief's durable-record argument ("the launch report is the durable
  record of every launch including pruned ones") was written to make deletion
  safe; keeping releases makes it unnecessary, not wrong. The ADR should say
  which paragraph it retires.
- itd-108's pinned archive and the bootstrap's `releases/latest` are
  unaffected by keeping old releases; they are affected by a deleted one.
- Immutable releases also stop a botched asset being replaced in place, which
  is already the documented position for the plugin zip (a fix is a new cut).

## itd-2609151658486398 — a cut publishes the advisories its fixes close

### Findings

- **The REST surface does everything the intent names.** `PATCH
  /repos/{owner}/{repo}/security-advisories/{ghsa_id}` takes `state`
  (`published`, `closed`, `draft`), `vulnerabilities[]` with
  `package.ecosystem` (`go` among the values, and `other`),
  `vulnerable_version_range` and `patched_versions`, and `credits[]` of
  `{login, type}`; "to add a credit, send the whole `credits` array". A CVE is
  requested with `POST .../{ghsa_id}/cve`, public repositories only. Tokens
  need `repository_advisories:write`. Listing filters by `state`, including
  `triage`. [EVIDENCE]
  [Repository security advisories REST](https://docs.github.com/en/rest/security-advisories/repository-advisories).
- **Credits are opt-in and the fix version must precede publication.**
  Publicly visible credits are those "the credited users have accepted";
  GitHub reviews a published advisory for the Advisory Database within "up to
  72 hours" and may send Dependabot alerts; without a fix version Dependabot
  alerts users "without offering any safe version to update to". A CVE request
  "doesn't make your security advisory public" and is usually reviewed within
  72 hours. [EVIDENCE]
  [About repository security advisories](https://docs.github.com/en/code-security/security-advisories/working-with-repository-security-advisories/about-repository-security-advisories),
  [Publishing a repository security advisory](https://docs.github.com/en/code-security/security-advisories/working-with-repository-security-advisories/publishing-a-repository-security-advisory).
- **Publication is one-way.** A published advisory is deleted only "by
  contacting Support". The publish step therefore belongs after the human
  release gate the pilot named (F-Q), never before it. [EVIDENCE]
  [Deleting a repository security advisory](https://docs.github.com/en/code-security/security-advisories/working-with-repository-security-advisories/deleting-a-repository-security-advisory).
- **"Not a vulnerability" is a close, with a comment first.** The maintainer
  flow for a triage report offers "Accept and open as draft" or "Close security
  advisory", and advises: "Where possible, you should add a comment explaining
  why you don't consider the report a security risk before you close the
  advisory." No close-reason enumeration is
  documented; the API's `state: closed` is the whole of it. [EVIDENCE]
  [Managing privately reported security vulnerabilities](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/fix-reported-vulnerabilities/managing-privately-reported-security-vulnerabilities).
- **OSV is the interchange shape the database exports.** Advisories are
  published "as JSON files in the Open Source Vulnerability (OSV) format";
  OSV's `fixed` event "is strongly preferred over `last_affected`", SEMVER
  ranges carry no leading `v`, and its credit types (FINDER, REPORTER,
  REMEDIATION_DEVELOPER, REMEDIATION_REVIEWER, REMEDIATION_VERIFIER, ...)
  mirror GitHub's. OpenSSF recommends "creating a JSON entry using the OSV
  schema and publishing this somewhere accessible over HTTP" inside its CVE
  step, and says omitting credit is "inappropriate" unless declined.
  [EVIDENCE] / [CONSENSUS]
  [About the GitHub Advisory Database](https://docs.github.com/en/code-security/security-advisories/working-with-global-security-advisories-from-the-github-advisory-database/about-the-github-advisory-database),
  [OSV schema](https://ossf.github.io/osv-schema/),
  [OpenSSF maintainer guide](https://github.com/ossf/oss-vulnerability-guide/blob/main/maintainer-guide.md).
- **The ledger has no link to the advisory.** Pilot F-E: three trackers, no
  id linkage; the v0.6.7 cut published its GHSAs by hand after the release.
  An automated publish needs the fixing record to carry the GHSA id, or the
  advisory to carry the record id, before the cut. [LOCAL]
  ([pilot note](2026-08-27-security-advisory-handling-pilot.md)).

### Options weighed

1. Forge API from `launch ship`, after the Release is published: for each
   draft advisory whose linked record shipped, set `patched_versions` to the
   derived version and `state: published`; for each linked to a won't-fix
   record, comment and `state: closed`. Credits are sent as recorded on the
   advisory (the maintainer's doing at triage), never invented by the cut.
2. Publish from a workflow step on the release event instead of from the
   verb: the same calls, but the secret lives in CI and a failed publish is
   invisible to the person at the terminal.
3. Hand step kept, with the cut printing the list to publish. Fails the
   intent's "no hand step decides which ones publish".

### Recommendation

**Path 1 — adopt the forge API as it is**, option 1, called from `launch
ship` (the title's "release workflow" is read as the verb, not as a CI step,
which option 2 rejects), with one record change: the issue record gains an
`advisory:` field holding the GHSA id (the pilot's F-E gap), set at triage
when the advisory is accepted as a draft; the field is an issue-schema
change, a prerequisite for the record-lint and schema owners. The
`ecosystem` for a Go binary whose importable packages are all `internal/` is
`go` with the module path, so a reviewed advisory can reach anyone who
vendors the module; `other` is the alternative if the record decides
importers do not exist. The calls go over the pinned HTTP client the update
package already has, or through `gh api`; no Go dependency.

### Local fit risks

- F-Q: publication is after the human release gate, and is itself irreversible;
  the verb must show the exact PATCH bodies before sending, under the same
  consent shape as the cut.
- Outbound text: the advisory body and the credits are public the moment they
  publish; the re-read rule for outward text applies to the composed body.
- The derived version is known only at ship (F-P); `patched_versions` cannot
  be stamped on the draft earlier without risking a wrong number.

## itd-149 — inbound advisories and release cuts for every managed repo

The pilot note is the acceptance-criteria source; the research below adds
what the forge and the disclosure bodies document that the pilot did by hand.

### Findings

- **Private vulnerability reporting has a full API surface.** Reporters use
  `POST /repos/{owner}/{repo}/security-advisories/reports`; maintainers list
  with `state=triage`, accept into a draft, open a temporary private fork
  (`POST .../forks`, asynchronous, "up to 5 minutes"), and collaborate through
  `collaborating_users` and `collaborating_teams`. OpenSSF warns that "private
  forks created as part of GitHub Security Advisory do not have access to
  integration like CI systems", so tests run locally, which is the pilot's
  worktree discipline (F-C). [EVIDENCE]
  [Repository security advisories REST](https://docs.github.com/en/rest/security-advisories/repository-advisories),
  [OpenSSF maintainer guide](https://github.com/ossf/oss-vulnerability-guide/blob/main/maintainer-guide.md).
- **The triage guidance asks the maintainer to look for an AI-assistance
  disclosure.** The flow says "Carefully review the report details and any
  disclosure of AI assistance"; no structured field is documented, so the
  disclosure is whatever the reporter volunteers, a provenance signal the
  pilot's F-H intake records when present rather than one it can rely on.
  [EVIDENCE]
  [Managing privately reported security vulnerabilities](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/fix-reported-vulnerabilities/managing-privately-reported-security-vulnerabilities).
- **The credit types cover three of the pilot's four roles.** GitHub's
  `credits.type` enum has ten values: `analyst`, `finder`, `reporter`,
  `coordinator`, `remediation_developer`, `remediation_reviewer`,
  `remediation_verifier`, `tool`, `sponsor` and `other`. F-I's reporter maps
  to `finder` or `reporter`, the author of record to
  `remediation_developer`, and the independent gate to
  `remediation_reviewer` or `remediation_verifier`; the authoriser has no
  direct type, with `coordinator` and `other` the candidates. The advisory
  can carry most of the roles the commit trailers carry, and the mapping is
  the record's to fix. [EVIDENCE]
  [Repository security advisories REST](https://docs.github.com/en/rest/security-advisories/repository-advisories).
- **Timelines have reference points.** OpenSSF: acknowledge "quickly",
  "say within 1-2 days"; embargo defaults of under 7 days (distros list),
  45 days (CERT/CC) and 90 days (Google), "90 days is the longest"; publish
  the OSV entry with the CVE; withholding exploit details helps "for a few
  days at most"; a partial fix that breaks the exploitation chain beats none
  and is documented as partial. The repository's `SECURITY.md` promises an
  acknowledgement within a week, slower than the guide's 1–2 day target; the
  7, 45 and 90 days are embargo defaults, not acknowledgement times.
  [CONSENSUS]
  [OpenSSF maintainer guide](https://github.com/ossf/oss-vulnerability-guide/blob/main/maintainer-guide.md).
- **Only reviewed advisories drive Dependabot.** Unreviewed advisories are
  imported automatically from the NVD feed and raise no alerts; a repository
  advisory becomes reviewed through GitHub's post-publication review. [EVIDENCE]
  [About the GitHub Advisory Database](https://docs.github.com/en/code-security/security-advisories/working-with-global-security-advisories-from-the-github-advisory-database/about-the-github-advisory-database).
- **Attribution must read the running model.** Pilot F-G is unchanged by any
  source opened here; nothing in the forge records which model assisted a fix,
  so the transcript remains the only authority. [LOCAL]

### Options weighed

1. Native loop over the pilot's protocol, with the forge calls behind one
   adapter and the review stages delegated to host agents.
2. A hosted workflow (GitHub Actions) running the loop: fails host-delegated
   review and the human-by-design gates (F-Q), and gives the secret to CI.
3. Adopt an external triage product. None surveyed covers the ledger, the
   worktree fix stage or the derived cut; rejected without naming one.

### Recommendation

**Path 2 — native, shaped to the OpenSSF maintainer flow and the forge's
advisory API, with the forge adapter as the seam.** Option 1. The loop's
stages are the pilot's; the research fixes three of its shapes: intake
records an AI-assistance disclosure as provenance where the report volunteers
one; credits are filed by the matching types, with the authoriser's type left
to the record; and the two human gates (PROMOTE and publish) stay human.
itd-2609151658486398 is the last stage of this loop and ships first. No new
Go dependency.

### Local fit risks

- Agents stay out of the ledger (F-F): the forge adapter writes advisories,
  the parent writes records; a sub-agent that could PATCH an advisory could
  publish one.
- Outbound text: no unpublished GHSA id in a commit, PR or comment; the guard
  already judges `gh pr` and `gh issue` text, and should judge `gh api`
  advisory bodies the same way.
- Host-delegated review: the adversarial gate over the assembled diff (F-D)
  is a host agent, never an API call abcd makes itself.

## Not worth adopting

- **An offline signing key now** (minisign or cosign with a held key). It
  defends a forge compromise the record has classed as deferred, and it adds
  a key to keep and rotate; the pinned archive, attestations and immutable
  releases cover the threats named today.
- **Known Folders for the Windows binary.** Correct by Microsoft's own
  document, but every developer tool a person on Windows already has
  (the harness, uv, mise) uses `~\.local\bin`; following the Known Folder
  convention would put abcd off the PATH those tools created.
- **`~/Library/Application Support` on macOS.** Same reasoning: the field
  keeps dot-directories, the repository already has one resolver and a
  `.noindex` suffix for the Spotlight concern, and a move there would strand
  the records every existing checkout holds.
- **A hosted workflow for advisory handling or publication.** Puts the
  publish secret in CI, hides a failed publish from the person at the
  terminal, and cannot delegate review to the host.
- **Publishing the plugin through winget, npm or Homebrew as the first
  channel.** Each is a second downloader with its own update command; the
  starter intent exists to get to one. They are later channels behind the
  asset-layout seam (iss-380 for Homebrew).
- **Building the release prune** (itd-70 as drafted). See that section.

## Not verified

- Whether the Windows per-user `HKCU\Environment` key is the documented home
  of the user `Path`: the Win32 page opened documents the system key and the
  `WM_SETTINGCHANGE` broadcast; the per-user key was not quoted on it.
- The eligibility and price of Microsoft's Artifact Signing for an individual
  maintainer, and whether a signed binary from a new publisher still draws a
  SmartScreen warning: search results asserted both, but the pages were not
  opened and are not cited.
- Where winget installs a portable package and how it adds it to the PATH:
  the `install` reference names a portable rename option but not the
  location.
- Whether the harness passes its own running version to a hook's environment
  or input, which would make itd-2610031325050110's option 2 readable.
- The cargo-dist shell installer's checksum step and unmanaged mode: the
  project's book was unreachable (DNS) and the GitHub copy of its page does
  not cover them.
- Reports of AI-generated vulnerability-report floods (curl is the case named
  in search results): no primary page was opened, so the figures are not
  repeated here.
- The `arp242` "curl to sh" essay returned 404 in this run; the common
  position it is known for is cited above through the Kicksecure page instead.
- GitHub's "Verifying the integrity of a release" guide for the immutable
  release attestation: linked from the concept page but not opened.
- SLSA v1.2's threats page: not opened; the "does not protect against"
  wording above is the build-requirements page's.
