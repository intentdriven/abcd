---
schema_version: 1
id: "iss-205"
slug: "every-one-of-the-17-files-under-commands"
severity: "critical"
category: "bug"
source: "user-observation"
found_during: "first manual plugin install test (2026-08-10)"
found_at: "commands/"
related_intents: [itd-108]
related_issues: [iss-206]
resolution: "The command surface resolves the binary from ${CLAUDE_PLUGIN_ROOT} first, PATH second, and go run ./cmd/abcd only in a source checkout; a Go detector test under internal/core/launch fails any commands/ file that regresses the ladder."
impact: fix
---

Every one of the 17 files under commands/ resolves the binary as bare 'abcd' on PATH, with a documented fallback to 'go run ./cmd/abcd'. None of them mentions ${CLAUDE_PLUGIN_ROOT} — grep confirms zero occurrences across commands/. But itd-105 decided 'PATH: deferred to ahoy', so a by-the-book marketplace install provisions the binary ONLY into the plugin root and leaves nothing on PATH. The two halves therefore still do not meet: bootstrap installs a checksum-verified binary that no command can see. Observed on the first manual install (2026-08-10): 'which abcd' reports not found, and bare /abcd fell through to the source-build path — 54s, requiring a Go toolchain, and succeeding only because the marketplace clone happened to be on disk. On a clean machine without Go the entire plugin command surface is non-functional despite a healthy binary sitting in the plugin root. This directly contradicts itd-105's own premise: 'Nobody clones the repo to use abcd: the binary is all a user needs.' The fix is supported and cheap — the plugins reference confirms ${CLAUDE_PLUGIN_ROOT} substitutes in 'Skill and agent content: anywhere the placeholder appears'. Put "${CLAUDE_PLUGIN_ROOT}/abcd" first in the resolution order in all 17 command files, PATH second, 'go run' last.

**Prerequisite for itd-108 — fix this first, not alongside (2026-08-10).** itd-108 repoints the marketplace at the curated release artifact, whose payload is `.abcd/config/launch-payload.json`: `.claude-plugin`, `commands`, `agents`, `hooks`, `scripts`, `docs`, `README.md`, `LICENSE`. There is no `cmd/`, no `internal/`, and no `go.mod` in it. So the `go run ./cmd/abcd` fallback is not merely slow under itd-108 — it becomes impossible, because no source is present to build from. That fallback is currently the ONLY reason the command surface works on a fresh install (observed 2026-08-10: bare /abcd succeeded solely because the marketplace clone happened to carry `cmd/`). Ship itd-108 before this fix and every `/abcd:*` command goes from "54 seconds and needs Go" to a hard failure with no recovery path on the user's machine.

This inverts the severity relationship between the two records: today iss-205 is masked by an accident of packaging, and itd-108 removes the mask. Two consequences for whoever sequences the work. First, the `${CLAUDE_PLUGIN_ROOT}` fix must land and be verified on a real install BEFORE the payload slims, so there is never a release in which the fallback is gone and the primary path is not yet wired. Second, the third rung of the ladder should be reconsidered rather than merely demoted: under itd-108 a `go run` fallback can only fire for someone running from a source checkout, so retaining it as generic advice in a shipped command file would print an instruction that cannot work for the users those files reach. Deciding what the last rung says — a source-checkout-only note, or nothing — belongs with this fix, not with itd-108.