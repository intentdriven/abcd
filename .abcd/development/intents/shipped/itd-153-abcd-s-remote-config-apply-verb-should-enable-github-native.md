---
id: itd-153
shipped_in: v0.6.8
slug: abcd-s-remote-config-apply-verb-should-enable-github-native
spec_id: spc-46
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-2608270636272755
impact: additive
---

# abcd's remote-config apply verb should enable GitHub native secret scanning AND secret-scanning push protection by default on every managed repo (belt-and-braces alongside the CI gitleaks full-history scan): push protection blocks a secret at push time, earlier than CI, and secret scanning covers the default branch continuously. Enabled by hand on [redacted-user]/abcd 2026-08-27 via PATCH /repos/{o}/{r} security_and_analysis (both were disabled). Notes: push protection requires secret scanning enabled first; both are free on public repos; secret_scanning_non_provider_patterns and secret_scanning_validity_checks remain optional further hardening. Homes: the read-only VERIFY side is itd-92 (the doctor now reports these toggles); the APPLY-by-default belongs to the separate adr-44-bound apply intent itd-92 defers to, adjacent to itd-106 (abcd sets up the CI a repo requires); the desired state should be mirrored in the repo-settings.json sibling (iss-2608270512210664).

## Press Release

> _Seeded by promotion from iss-2608270636272755. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2608270636272755`: abcd's remote-config apply verb should enable GitHub native secret scanning AND secret-scanning push protection by default on every managed repo (belt-and-braces alongside the CI gitleaks full-history scan): push protection blocks a secret at push time, earlier than CI, and secret scanning covers the default branch continuously. Enabled by hand on [redacted-user]/abcd 2026-08-27 via PATCH /repos/{o}/{r} security_and_analysis (both were disabled). Notes: push protection requires secret scanning enabled first; both are free on public repos; secret_scanning_non_provider_patterns and secret_scanning_validity_checks remain optional further hardening. Homes: the read-only VERIFY side is itd-92 (the doctor now reports these toggles); the APPLY-by-default belongs to the separate adr-44-bound apply intent itd-92 defers to, adjacent to itd-106 (abcd sets up the CI a repo requires); the desired state should be mirrored in the repo-settings.json sibling (iss-2608270512210664).. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a managed repo whose GitHub native secret scanning and secret-scanning push protection are both disabled, **when** the remote config-apply verb runs with no opt-out, **then** both are enabled, and secret scanning is enabled before push protection because push protection requires it first.
- **Given** a managed repo whose config declares an opt-out for native secret scanning, **when** config-apply runs, **then** the toggles are left as they are and neither is enabled.
- **Given** a managed repo where both toggles are already enabled, **when** config-apply runs, **then** the verb is idempotent: it reports no change, makes no API write that alters state, and exits cleanly.
- **Given** config-apply has enabled the toggles, **when** the desired state is recorded, **then** it is mirrored in the repo-settings.json sibling so a later verify reads the same intended state.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-1ec1064a175f -->
Fidelity review — receipt rcp-1ec1064a175f (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:8cb523d9a8e58615fa6c7beeb832b8e87f0f157f9515c2bd015e5d95c10602e6
Input attestations: diff:328a6755^1..328a6755 (PR #555), judged against the tree at bad1c73e@-;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: RemoteApply reads the state, then PATCHes secret_scanning before secret_scanning_push_protection in a fixed ordered loop, and the fake-gh test asserts one GET then the two PATCHes in that order; concern: the write proceeds only after the caller confirms (Prompter, `--yes` on the CLI) per adr-44, and it is an explicit `ahoy remote apply` invocation, never run by `ahoy install`
  evidence: internal/core/ahoy/remote.go:284 — "for _, step := range []struct{ key, have string }{"
  evidence: internal/core/ahoy/remote.go:271 — "if !p.Confirm("Change GitHub settings on " + res.Repo"
  evidence: internal/core/ahoy/remote_test.go:191 — "func TestRemoteApplyEnablesSecretScanningBeforePushProtection"
  evidence: internal/core/ahoy/remote_test.go:136 — "func TestRemoteApplyRequiresConfirmationNotJustInvocation"
  evidence: internal/surface/cli/cli.go:2806 — "applyCmd.Flags().BoolVar(&remoteYes, "yes", false"
- ac-2 — MET: an explicit scan.native_secret_scanning=false short-circuits in remotePrepare with status opted_out before any gh call, and the test asserts zero remote calls
  evidence: internal/core/ahoy/remote.go:352 — "if optedOut {"
  evidence: internal/core/ahoy/remote.go:138 — "v, ok := boolVal(subMap(cfg, "scan"), "native_secret_scanning")"
  evidence: internal/core/ahoy/remote_test.go:323 — "func TestRemoteApplyHonoursTheOptOut"
- ac-3 — MET: an already-enabled toggle is skipped in the loop, no PATCH is made, Changes stays empty, and a re-run reports already_up_to_date with an unchanged tree; the first run may still write the in-tree mirror, which is not a remote write
  evidence: internal/core/ahoy/remote.go:288 — "if step.have == statusEnabled {"
  evidence: internal/core/ahoy/remote.go:312 — "res.Status = "already_up_to_date""
  evidence: internal/core/ahoy/remote_test.go:281 — "func TestRemoteApplyIsIdempotent"
- ac-4 — MET: after every needed enable succeeds the verb writes .abcd/work/rulesets/repo-settings.json recording both toggles enabled under `managed`, and the repository's own mirror is present in the tree with that shape
  evidence: internal/core/ahoy/remote.go:26 — "const RepoSettingsMirrorRelPath = ".abcd/work/rulesets/repo-settings.json""
  evidence: internal/core/ahoy/remote.go:516 — "secretScanningKey: map[string]string{"status": statusEnabled},"
  evidence: internal/core/ahoy/remote_test.go:226 — "func TestRemoteApplyMirrorsTheDesiredState"
  evidence: .abcd/work/rulesets/repo-settings.json:4 — ""secret_scanning": {"

Gap audit:
- honoured:
  - the verb is wired on both surfaces: `abcd ahoy remote [apply]` on the CLI and the plugin command page
    evidence: internal/surface/cli/cli.go:2750 — "Use: "remote","
    evidence: commands/ahoy.md:214 — "ahoy remote apply --json"
  - the write goes through `gh` with an explicit hostname, inheriting the caller's identity
    evidence: internal/core/ahoy/remote.go:443 — ""api", "--hostname", githubHost, "--method", "PATCH","
  - a failed enable stops the sequence and records no desired state
    evidence: internal/core/ahoy/remote.go:293 — "the remaining steps were not attempted and no desired state was recorded"
    evidence: internal/core/ahoy/remote_test.go:508 — "func TestRemoteApplyStopsAtTheFirstFailedWrite"
- diverged:
  - the intent title promises enabling 'by default on every managed repo'; delivered as an explicit, confirmed verb that `ahoy install` never runs — the spec's Decisions record this as the adr-44 ruling
    evidence: internal/core/ahoy/remote.go:242 — "a remote write happens only through a dedicated verb the user invokes AND CONFIRMS"
    evidence: .abcd/development/specs/closed/spc-46-abcd-s-remote-config-apply-verb-should-enable-github-native.md:96 — "the verb acts only on explicit invocation against a managed repo"
- missing: (none)