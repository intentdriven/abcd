---
id: itd-154
shipped_in: v0.6.8
slug: s4-gate-failed-bootstrap-provisioned-no-binary
spec_id: spc-47
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-253
impact: fix
---

# The Cut A §4 manual gate FAILED on assertions 3 and 4 (transcript: .abcd/.work.local/scratch/2026-08-16-s4-transcript.txt, machine: a no-Go Mac with a fresh harness install). After /plugin marketplace add + /plugin install, NO bootstrap line ever appeared at any of three session starts, the plugin root (…/plugins/cache/abcd-marketplace/abcd/<sha>/) contained a full source checkout but no provisioned binary, and every UserPromptSubmit and PreToolUse hook errored 'No such file or directory' for the entire evening — the exact failure family the gate checks for zero of. The command surface functioned only via rung two (the PATH binary the separate one-liner installed), so the no-Go promise is unproven; ahoy install itself reports plugin.root_missing as resolvable:false. Assertions 1, 2 and 5 passed (checksum-verified one-liner with no sudo, abcd v0.5.0; transcript captured; non-interactive install with a clean, path-scrubbed receipt — including a correct abort in a non-repo directory). Per the gate's own rule this BLOCKS Cut B. Environment notes for reproduction: the harness never visibly ran the plugin's SessionStart chain after install (bootstrap.sh emitted nothing — neither success nor refusal, despite loud-staging), and the machine's home dirs are cloud-synced (anomalous stat results, 65535 link counts, multi-minute tree walks in the clone). Detector: the §4 checklist; acceptance: a fresh-machine plugin install yields one bootstrap success line, a provisioned plugin-root binary, and /abcd answering in about a second with no Go.

## Press Release

> _Seeded by promotion from iss-253. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-253`: The Cut A §4 manual gate FAILED on assertions 3 and 4 (transcript: .abcd/.work.local/scratch/2026-08-16-s4-transcript.txt, machine: a no-Go Mac with a fresh harness install). After /plugin marketplace add + /plugin install, NO bootstrap line ever appeared at any of three session starts, the plugin root (…/plugins/cache/abcd-marketplace/abcd/<sha>/) contained a full source checkout but no provisioned binary, and every UserPromptSubmit and PreToolUse hook errored 'No such file or directory' for the entire evening — the exact failure family the gate checks for zero of. The command surface functioned only via rung two (the PATH binary the separate one-liner installed), so the no-Go promise is unproven; ahoy install itself reports plugin.root_missing as resolvable:false. Assertions 1, 2 and 5 passed (checksum-verified one-liner with no sudo, abcd v0.5.0; transcript captured; non-interactive install with a clean, path-scrubbed receipt — including a correct abort in a non-repo directory). Per the gate's own rule this BLOCKS Cut B. Environment notes for reproduction: the harness never visibly ran the plugin's SessionStart chain after install (bootstrap.sh emitted nothing — neither success nor refusal, despite loud-staging), and the machine's home dirs are cloud-synced (anomalous stat results, 65535 link counts, multi-minute tree walks in the clone). Detector: the §4 checklist; acceptance: a fresh-machine plugin install yields one bootstrap success line, a provisioned plugin-root binary, and /abcd answering in about a second with no Go.. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a machine with no Go toolchain and no abcd binary present, **when** the plugin is installed and a session starts, **then** bootstrap downloads the release binary built for that platform and `/abcd` answers in about a second.
- **Given** the bootstrap provisioning step runs, **when** it fetches the binary, **then** it emits a visible, loud-staged `provisioning the abcd binary…` line and never proceeds silently.
- **Given** a binary has been downloaded, **when** bootstrap installs it into the plugin root, **then** the binary is verified against the release checksum before it is used.
- **Given** the download fails or the checksum does not match, **when** bootstrap runs, **then** it fails loudly with a clear message rather than leaving the UserPromptSubmit and PreToolUse hooks in a limping `No such file or directory` state.
- **Given** the plugin release payload, **when** it is inspected, **then** it bundles no platform binaries; the binary arrives only via the checksum-verified download.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-56d6066b23e8 -->
Fidelity review — receipt rcp-56d6066b23e8 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:f68931220fa803a190338de3a0c05e66e40bf3d520835267262e85753b20bd52
Input attestations: diff:328a6755^1..328a6755 (PR #555), judged against the tree at bad1c73e@-;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 1 · NOT_MET 1 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: the self-check runs the shipped script under a base-system PATH with no Go against a fixture release serving the real built binary, then asserts the provisioned binary answers `version`; concern: the 'about a second' half is deliberately not asserted (a duration is a host property; the earlier budget flaked on a loaded runner)
  evidence: internal/surface/cli/bootstrap_freshinstall_test.go:49 — "const bootstrapGoFreePath = "/usr/bin:/bin:/usr/sbin:/sbin""
  evidence: internal/surface/cli/bootstrap_freshinstall_test.go:103 — "func TestBootstrapFreshInstallSelfCheck"
  evidence: internal/surface/cli/bootstrap_freshinstall_test.go:147 — "asserts what the binary does and never how long the spawn took"
- ac-2 — NOT_MET: promised: a visible `provisioning the abcd binary…` line whenever bootstrap fetches the binary; delivered: the script deliberately prints NO eager provisioning line — the phrase appears only in the EXIT trap's report of a run that died — so a successful fetch emits only the terminal `abcd bootstrap: installed` line; the 'never proceeds silently' half is honoured by the staged/terminal contract
  evidence: hooks/bootstrap.sh:519 — "There is deliberately NO eager "provisioning…" line printed here, loud as"
  evidence: hooks/bootstrap.sh:257 — "say_refusal 'provisioning the abcd binary for this plugin root ended without installing it"
  evidence: internal/surface/cli/bootstrap_freshinstall_test.go:117 — "the success must be the FIRST line of stderr, the only one the transcript renders"
  evidence: internal/surface/cli/bootstrap_freshinstall_test.go:248 — "if got := firstLine(out); !strings.Contains(got, "provisioning the abcd binary")"
- ac-3 — MET: the download is checked against the release checksums.txt and refused on mismatch, and the hash is re-verified at promotion into the plugin root; a tampered manifest fails loudly and leaves no binary
  evidence: hooks/bootstrap.sh:700 — "refuse "the downloaded $asset does not match its SHA-256 checksum in the release checksums.txt"
  evidence: hooks/bootstrap.sh:842 — "[ "$got_sha" = "$expected_sha" ] ||"
  evidence: internal/surface/cli/bootstrap_freshinstall_test.go:179 — "func TestBootstrapFailsLoudlyAndLeavesNothingHalfInstalled"
- ac-4 — MET: every failing path goes through refuse()/say_refusal with the three ways out, and the EXIT trap converts a mid-provision death into the same refusal, releasing the lock and leaving no binary; both shapes are tested
  evidence: hooks/bootstrap.sh:211 — "refuse() {"
  evidence: hooks/bootstrap.sh:256 — "if [ -n "$staged" ] && [ -z "$terminal" ]; then"
  evidence: internal/surface/cli/bootstrap_freshinstall_test.go:223 — "func TestBootstrapConvertsASilentDeathIntoARefusal"
- ac-5 — MET: the launch bundler rejects any basename matching a built abcd binary with a dedicated RejectedPlatformBinary reason, and two tests pin that the payload never ships one and names no binary directory
  evidence: internal/core/launch/bundle.go:58 — "RejectedPlatformBinary RejectedReason = "platform_binary""
  evidence: internal/core/launch/bundle.go:913 — "var platformBinaryRe = regexp.MustCompile(`^abcd(-[a-z0-9]+-[a-z0-9]+)?(\.exe)?$`)"
  evidence: internal/core/launch/platform_binary_test.go:21 — "func TestBundleNeverShipsAPlatformBinary"
  evidence: internal/core/launch/platform_binary_test.go:63 — "func TestCommittedPayloadNamesNoBinaryDirectory"

Gap audit:
- honoured:
  - once provisioning begins the run owes exactly one terminal line, success or refusal, and a silent death is converted into a refusal
    evidence: hooks/bootstrap.sh:168 — "staged records that provisioning BEGAN, and terminal that the run has already"
    evidence: hooks/bootstrap.sh:244 — "on_exit is the EXIT trap: it cleans up, and it converts a SILENT death into a"
  - the §4 checklist is an automated gate run under a Go-free PATH
    evidence: internal/surface/cli/bootstrap_freshinstall_test.go:100 — "TestBootstrapFreshInstallSelfCheck is the §4 checklist as a gate"
  - download-only: the plugin payload bundles no platform binary
    evidence: internal/core/launch/bundle.go:264 — "A released platform binary never ships in the payload"
- diverged:
  - the promised loud-staged `provisioning the abcd binary…` line on every fetch was replaced by a single terminal line, with the provisioning phrase reserved for the death report — reasoned in place (only the first stderr line reaches the transcript) but not what ac-2 states
    evidence: hooks/bootstrap.sh:520 — "Only the FIRST line of a hook's stderr reaches the"
    evidence: hooks/bootstrap.sh:527 — "announcement is therefore held and spent only where it is the only thing a"
  - the 'answers in about a second' timing is not asserted by the gate
    evidence: internal/surface/cli/bootstrap_freshinstall_test.go:150 — "the five-second budget that once stood here, widened for"
- missing: (none)