---
id: itd-154
slug: s4-gate-failed-bootstrap-provisioned-no-binary
spec_id: spc-47
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-253
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

_Empty. Populated by intent-auditor when intent moves to shipped/._
