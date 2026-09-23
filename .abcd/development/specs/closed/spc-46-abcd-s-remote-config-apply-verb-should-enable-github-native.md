---
id: spc-46
slug: abcd-s-remote-config-apply-verb-should-enable-github-native
intent: itd-153
---
# abcd-s-remote-config-apply-verb-should-enable-github-native

## Summary

Teaches abcd's remote config-apply verb to enable GitHub native secret scanning
and secret-scanning push protection by default on a managed repo — belt-and-braces
alongside the CI gitleaks full-history scan — with a config opt-out, idempotent
no-op when already enabled, and the desired state mirrored into a
`repo-settings.json` sibling so a later verify reads the same intent. Secret
scanning is enabled before push protection because GitHub requires it first.

## Scope

In:

- A remote config-apply verb wired into `internal/surface/cli/cli.go`'s
  `newAhoyCommand` (line 1826), alongside the existing `install`/`doctor`
  subcommands.
- A new core file under `internal/core/ahoy/` performing the GitHub API write to
  `security_and_analysis`, plus an opt-out flag on the config model and an
  idempotency check.
- A `repo-settings.json` mirror writer (sibling to `.abcd/work/rulesets/`),
  written via `fsutil.WriteFileAtomic`.
- Tests modelled on `internal/core/ahoy/idempotency_test.go` and `apply_test.go`.

Out:

- `secret_scanning_non_provider_patterns` and `secret_scanning_validity_checks`
  (optional further hardening) — not enabled by this verb.
- Branch-protection/ruleset drift (itd-92's read-only VERIFY remit) and CI setup
  (itd-106) — adjacent, not in scope here.
- No uninvited remote mutation: the verb answers to adr-44 (see Decisions).

## Approach

**The write.** The change is net-new — no Go code today touches
`security_and_analysis`, and there is no GitHub API wrapper (the only shell-outs
are `git`). Following the existing `exec.Command` precedent (`ahoy/store.go:53`),
the verb performs the enable via `gh api --method PATCH /repos/{owner}/{repo}`
with a `security_and_analysis` body, in two ordered steps: first
`secret_scanning: {status: enabled}`, then
`secret_scanning_push_protection: {status: enabled}` — the ordering is
load-bearing because push protection requires secret scanning enabled first.
Reusing `gh` (rather than a raw `net/http` client) inherits the caller's
authenticated identity, which adr-44 requires.

**The opt-out.** The config model `InstallConfig` (`ahoy/ahoy.go:100`) already
uses the nil-is-unset `*bool` pattern (`ScanDeep *bool`). A sibling
`NativeSecretScanning *bool` follows it: nil or true → enable; an explicit
`false` in `.abcd/config.json` is the opt-out, and the verb leaves both toggles
untouched.

**Idempotency.** Before any write, the verb reads the current
`security_and_analysis` state via `gh api GET /repos/{owner}/{repo}`. When both
toggles are already `enabled`, it makes no PATCH, reports no change, and exits
cleanly — mirroring `Install`'s documented "writes nothing, reports
already_up_to_date" precedent (`apply.go:17–19`) and surfaced through the same
`InstallResult{Status, Changes, Notes}` shape (`ahoy.go:132`). A loud refusal for
an unresolvable owner/repo or an auth failure uses the existing `refuse`
mechanism (`apply.go:290`).

**The mirror.** The desired state is written to a `repo-settings.json` sibling of
`.abcd/work/rulesets/` (whose README already notes the drift check is itd-92's
remit and the mirror is hand-refreshed until it ships). The verb records
`secret_scanning: enabled` and `secret_scanning_push_protection: enabled` (and
the opt-out when set) via `fsutil.WriteFileAtomic`, so a later verify reads the
same intended state the apply wrote.

## How it satisfies each acceptance criterion

- *Both disabled, no opt-out → both enabled, secret scanning first* — the two
  ordered PATCH steps. Test: a fake `gh` seam records the call order and asserts
  `secret_scanning` is enabled before `secret_scanning_push_protection`.
- *Config declares an opt-out → toggles left untouched* — the
  `NativeSecretScanning == false` branch short-circuits before any write. Test:
  with the opt-out set, assert zero PATCH calls.
- *Both already enabled → idempotent no-op, no state-altering write, clean exit* —
  the pre-write GET short-circuits. Test: seed the GET to report both enabled and
  assert no PATCH, `Status` reports no change, exit clean (mirrors
  `idempotency_test.go`).
- *Desired state mirrored into `repo-settings.json`* — the mirror writer. Test:
  after apply, read `repo-settings.json` and assert it records both toggles
  enabled so a verify reads the same state.

## Decisions

Enable-by-default with an explicit opt-out, over opt-in: the security posture is
the intended default for a managed repo (both are free on public repos), and the
`*bool` nil-is-unset pattern already established for `ScanDeep` gives a clean
opt-out without a tri-state flag. The write goes through `gh` to inherit the
caller's identity and honour adr-44's "never mutate a remote uninvited" rule —
the verb acts only on explicit invocation against a managed repo, and records
what it changed. The read-only VERIFY of these toggles remains itd-92's remit;
this verb is the APPLY side it defers to.
