---
id: spc-2609230613208843
slug: per-platform-staleness-calibration
intent: itd-111
origin: researcher-authored
production_mode: hand-written
---
# Staleness behaviour is calibrated per platform

## Summary

The remainder of [itd-111](../../intents/shipped/itd-111-a-stale-abcd-never-answers-silently-every-surface-that-runs.md)
that [spc-22](spc-22-a-stale-abcd-never-answers-silently-every-surface-that-runs.md) did not deliver. spc-22 closed on
2026-09-23 with acceptance criteria 1 to 7 delivered: the SessionStart staleness notice, the install refusal on a stale or unknown vintage, install mode and vintage on `version` and `ahoy`, no version-discovery network request without `version --check`, the transition report, and the unknown-never-fresh outcome. This spec carries what did not ship.

The delivered part was already announced in the [0.5.0] changelog section,
and the intent carries no `shipped_in:` because it stays planned. The close
that ships the intent through this spec is the one that decides how the cut
reports it: without `shipped_in:` the whole intent is announced again, which
is the redundant line the changelog composer prefers to a silent omission.

## Scope

- **Acceptance criterion 8 (missing).** Given the same staleness scenarios on macOS and on Linux, when the itd-109 calibration runs, then observed behaviour matches, recorded as a machine-class criterion, human-verified per platform. No recorded macOS or Linux human-verified calibration run exists on `main` at closure.

## How the criterion is satisfied

The product thinker ruled on 2026-09-29 that an agent's check on macOS plus the
Linux CI run stand in for the per-platform check by a person, and that the
intent may be finished. The criterion is amended to match in the intent, which
states the ruling. itd-109 is still a draft, so the calibration harness the
original wording named does not exist; the scenarios run through the tests and
binaries that already carry them.

**Linux: CI.** The push run on `main` at de42f275a (the merge of pull request
#751), CI workflow run 36608836118, job `check (ubuntu-latest)` (job
109544802516), built with go1.26.7 linux/amd64 and passed
`internal/core/vintage`, `internal/core/ahoy` and `internal/surface/cli` under
`go test ./...` and again under the race lane. None of the staleness tests below
skips on Linux.

**macOS: the agent's check, 2026-09-29, at the same commit (darwin/arm64).**

1. The 25 staleness tests below passed under the declared toolchain (go1.26.7)
   and under the local go1.27.1, zero skipped, in 4.7 seconds wall time with
   `-p 4`: `TestOfReaderReadsABinaryWithoutRunningIt`,
   `TestCheckoutTipProvider`, `TestPinnedVersionProvider`,
   `TestGitHubReleaseFetcherReadsTagFromRedirect`,
   `TestGitHubReleaseFetcherNoTagIsAnError`, `TestReleaseProviderComparison`,
   `TestCompareOutcomes`, `TestCompareReportCarriesValues`,
   `TestCurrentFromSettings` (vintage); `TestVintageFromDogfoodCheckout`,
   `TestVintageFromUnknownCurrentIsTerminal`, `TestVintageFromPinnedInstall`,
   `TestVintageDisplayAndStaleness`, `TestReadPinnedTagPrefersRootThenCache`,
   `TestDogfoodStalenessFrom`, `TestInstallRefusesUnknownVintage`,
   `TestInstallRefusesStaleAgainstTip`,
   `TestInstallOverrideProceedsThroughStaleBinary`,
   `TestInstallProceedsThroughFreshBinary` (ahoy);
   `TestOnlyUpdateCheckTouchesTheNetwork`, `TestUpdateCheckNamesTheNextStep`,
   `TestSessionStartReportsVersionTransition`,
   `TestSessionStartSilentWhenVersionMatches`, `TestFormatStalenessNotice`,
   `TestModeAndStatuslineTouchNoNetwork` (cli).
2. End to end, on a scratch copy of that tree committed into its own
   repository, with a VCS-stamped binary built from it and a throwaway home
   folder:
   - **Fresh.** `abcd --version` and `abcd ahoy` print the binary's revision
     and `up to date`; `abcd hook session-start` gives no staleness notice.
   - **Stale** (one commit added after the build). Both print
     `stale — behind the checkout tip` naming the tip; the session-start notice
     names the binary, the commit it was built from, the tip, and
     `make build`; `abcd ahoy install` refuses with the mismatch named, and
     neither the home folder nor the working tree changed.
   - **Unknown** (a dirty rebuild, `vcs.modified=true`, and a
     `-buildvcs=false` build). Both print `unknown`, never `up to date`; the
     session-start notice names the dirty rebuild; `abcd ahoy install` refuses
     naming the rebuild fix, writing nothing.
   - **Explicit check.** `abcd update --check` fetched the latest release once
     and reported it with its source named (the check `abcd version --check`
     became when the verbs consolidated, itd-2609212130136102).
   Each surface answered within 0.4 seconds; the session-start hook, which
   also drains transcripts, within 0.2 seconds.

The observed behaviour matches across the two platforms. The behaviour itself
shipped in v0.5.0 and was announced there, so the intent carries
`shipped_in: v0.5.0`: this close adds verification and no new behaviour, and an
announcement at the next cut would repeat the 0.5.0 line.
