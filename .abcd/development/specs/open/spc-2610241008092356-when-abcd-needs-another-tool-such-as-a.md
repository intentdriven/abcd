---
id: spc-2610241008092356
slug: when-abcd-needs-another-tool-such-as-a
intent: itd-2610240918594138
origin: researcher-authored
production_mode: dictated-and-formatted
---
# when-abcd-needs-another-tool-such-as-a

## Summary

spc-2610241008092356 replaces the tool registry's Homebrew step with a checked
download pinned in abcd's own release, so setup installs gitleaks and gh for
the person alone, into their own bin folder, with no admin rights, and says
how far each download was checked (itd-2610240918594138, dcn-2610240845548776,
adr-2610241057124151). It resolves iss-2610240824571059.

## Scope

- In: the release table (gitleaks v8.30.1, gh v2.103.2, macOS and Linux on
  arm64 and amd64); the bounded https download from GitHub's release hosts;
  the sha256 check against the pin; the in-process signed build record check
  for gh; the archive read; the atomic write into `$XDG_BIN_HOME`, else
  `$XDG_DATA_HOME/../bin`, else `~/.local/bin`; the installed-tools record in
  `~/.abcd.noindex/tools.json`; abcd's callers running the recorded copy;
  uninstall removing recorded copies; the plan that decides whether a copy
  already installed is used; the minimum wiring into ahoy's dependency step,
  the gh offer, and ahoy uninstall.
- Out: the setup-flow redesign that lays the plan out before asking (a
  separate lane calls `Installer.Plan`); release-time checks that move a pin
  to a newer verified version (iss-2610240959030370, left open); Windows;
  tools whose makers publish no checksums.

## Approach

The table lives in `internal/core/tools/release.go`: per tool a `Release`
(repository, pinned version, an `Asset` per `<GOOS>/<GOARCH>` holding the full
`https://github.com/<repo>/releases/download/<tag>/<file>` URL, its sha256, the
archive kind and the program's member path), an optional `Signer` (repository,
workflow file, ref; gh is `cli/cli` `.github/workflows/deployment.yml` on
`refs/heads/trunk`, gitleaks has none), and a `Compatible` version range. The
digests were copied on 2026-10-24 from each release's published checksums file
(`gitleaks_8.30.1_checksums.txt`, `gh_2.103.2_checksums.txt`) and each matched
the downloaded asset. No version is ever looked up (invariant 7).

`Installer.Plan(name)` decides, from the disk, PATH and the version a copy
already installed reports through the tool's verify argv, with no network:
`use-abcd-copy` (an intact recorded copy), `use-existing` (a copy installed
some other way whose version lies in `Compatible`), `install` (abcd's own copy
offered, saying which of the two will run in abcd and in the person's shell),
or `unavailable` (no download for the platform, or the bin folder holds a
program of that name abcd did not install). `Installer.Install` refuses in CI,
uses the plan, asks the front door's `Confirm` with the plan in the
explanation, then downloads (`fetch.go`: https only, GitHub release hosts only,
no proxy, CA overrides held out of the root pool, urlguard on dial and
redirect, 256 MiB and five-minute bounds), checks the sha256, checks the
signed build record (`attest.go`: GitHub's attestations API, inline bundle or
`bundle_url` into GitHub's attestation store decoded from snappy by
`snappy.go`, verified with sigstore-go against the TUF-fetched trust root, the
exact SAN, issuer, source repository, ref and GitHub-hosted runner), reads the
member (`archive.go`), and only then writes with `fsutil.WriteFileAtomic`,
verifies the pinned version, and records path and digest (`local.go`). A
failed verify or record removes the copy again.

`tools.Resolve(name)` is where abcd runs a registered tool from: the recorded
copy while its bytes hash to the record, else PATH. The gitleaks adapter's
`NewDefault`, `internal/core/github` and `internal/core/landing` resolve gh and
gitleaks through it; ahoy's presence check (`onPath`) counts the recorded copy.
`tools.Uninstall(home)` removes recorded copies whose digest still matches and
drops the record; `ahoy.Uninstall` calls it and reports `tools` in its
receipt.

### Core API for the setup-flow lane

All in `internal/core/tools`, none of it prints:

- `tools.Default(guard string) *tools.Installer` — the production installer.
- `(*Installer).Plan(name string) tools.Plan` — what setup would do, with
  `Action` (`UseOwned`, `UseExisting`, `InstallOwn`, `Unavailable`),
  `Pinned`, `Compatible`, `Download` (version, URL, sha256, member, dest,
  `Checks`, `ChecksumOnly`), `Existing` and `Owned` (`Copy`: path, version,
  compatible, why), `Runs` and `Why`.
- `(*Installer).Install(name string, c tools.Capability, confirm tools.Confirm) tools.Result`
  — `Ready`, `Ran`, `Installed`, `Verified`, `Declined`, `Path`, `Version`,
  `Checked`, `ChecksumOnly`, `Runs`, `Why`, `OnDecline`, and `Summary()`.
- `tools.Explain(name, c) tools.Explanation` — now carrying `Download` and,
  when Install hands it to `Confirm`, `Plan`; `StepText()` and
  `ManualText()` (the by-hand URL, sha256 and destination).
- `(tools.Release).Checks() string` — how far a download is checked.
- `tools.Resolve(name string) (string, error)`; `tools.Uninstall(home string) ([]tools.Removal, error)`.

### How the compatible range prepares for iss-2610240959030370

The range is the one statement of which versions abcd works with, and
`TestEveryPinnedReleaseIsComplete` holds every pin inside it. A release-time
check that tries a newer release against abcd's gates would move the pin, its
digests and, when the gates show it, the range, in one change to
`release.go`; holding the last passing pin is leaving the table as it is. That
check is not built here.

## Acceptance-criteria satisfaction

1. Missing tool installed, pinned, into the person's folder, checked before
   writing, no admin rights, no admin step shown: `Installer.Install` order
   in `install.go`; `TestInstallDownloadsChecksWritesAndRecords`,
   `TestExplainNamesTheCheckedDownload` and `noAdmin` (no `sudo`, `brew`,
   `chown` in any explanation or result), `TestEveryPinnedReleaseIsComplete`,
   `TestBinDirFollowsUvsOrder`; at ahoy `TestDependencyYesRunsTheStepAndReportsIt`.
2. A failed check writes nothing and says why; checksum-only is said:
   `TestAFailedCheckWritesNothingAndSaysWhy` (tampered sha256, failed build
   record, failed download), `TestAFailedVerifyRemovesTheCopy`,
   `TestHowEachToolIsCheckedIsSaid`, `TestGhsRecordedBuildRecordVerifiesAgainstItsPinnedSigner`
   and `TestABuildRecordFromAnotherSignerOrForOtherBytesFails` over the
   recorded real gh v2.103.2 bundles.
3. A compatible copy is used and nothing installed; otherwise abcd's copy is
   offered, saying which runs, and the other copy is never changed:
   `TestACompatibleCopyIsUsedAndNothingInstalled`,
   `TestAnIncompatibleCopyGetsAbcdsOwnAndSaysWhichRuns`,
   `TestABinFolderCopyAbcdDidNotInstallIsNeverReplaced`; at ahoy
   `TestAnIncompatibleGitleaksIsAGapOfItsOwn`.
4. abcd runs its copy whatever PATH says, and uninstall removes only its
   copies: `TestResolvePrefersAbcdsCopyWhateverPathSays`,
   `TestUninstallRemovesOnlyAbcdsIntactCopies`; at ahoy
   `TestUninstallRemovesTheToolCopiesAbcdRecorded`.

Scope conditions: the four platforms are pinned for both tools; nothing needs
admin rights, so an account with or without them takes the same route; both
tools publish checksums.

Known limits: gh's offer at `ahoy remote apply` and `site setup` still counts
any gh on PATH as present (only a missing gh is offered abcd's copy), because
gh's compatible floor (2.102.0, for `gh attestation verify --source-ref`)
guards abcd's by-hand fallback, not `gh api`; the full criterion-3 judgement
runs for gitleaks at `ahoy install`. When the bin folder already holds a
foreign program of the tool's name, abcd installs nothing rather than choose
another folder.

## Footprint

- packages: internal/core/tools, internal/core/ahoy, internal/adapter/gitleaks,
  internal/core/github, internal/core/landing, internal/surface/cli
- docs: commands/ahoy.md, .abcd/development/brief/04-surfaces/01-ahoy.md,
  .abcd/development/decisions/adrs/2610241057124151-abcd-installs-the-tools-it-needs-from.md
- tests: internal/core/tools/tools_test.go (and testdata/attest),
  internal/core/ahoy/tools_route_test.go, internal/core/ahoy/remote_gh_offer_test.go,
  internal/core/ahoy/remote_gh_explain_test.go, internal/core/site/setup_test.go,
  internal/surface/cli/ahoy_tool_confirm_test.go,
  internal/surface/cli/ahoy_remote_gh_offer_test.go,
  internal/surface/cli/site_setup_test.go

## Steps

_No steps listed: the spec is built as one step. To split it, list the steps in order as `1. <title>`, each with `- packages:` and `- tests:` indented beneath it; `- landed: <pull request or commit>` marks a step that has landed._
