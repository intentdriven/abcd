---
schema_version: 1
id: "iss-2608300930057882"
slug: "intent-ready-grounds-json-omits-the-promised-members"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "itd-179 adversarial ruthless review, 2026-08-30"
found_at: "commands/intent.md, internal/surface/cli/cli.go (intent ready --grounds), internal/core/intent/grounds.go (ParseGrounds, RecordGrounds)"
resolution: "intent ready --grounds --json now emits an envelope carrying the grounds write beside the readiness report, and the write receipt is printed before the gate is consulted so a later fault cannot hide it; ParseGrounds applies the substance floor the seventh check claims; RecordGrounds refuses a shipped or superseded record, so the forward-only exemption is a true statement about the corpus; and every grounds refusal on the capture routes exits 2 through one core sentinel, missing and malformed alike."
impact: fix
---

itd-179 ruthless findings: the plugin surface instructs the host to report a redacted member from intent ready --grounds --json that the verb never emits — the redaction note goes to stderr and the JSON is the unchanged readiness result, so the redaction-is-never-silent promise is broken on the one plane the feature is wired for (render an envelope carrying the grounds result beside the readiness result); the readiness reader admits a hand-typed entry below the writer's substance floor, so the seventh check under-enforces its own claim (skip sub-floor bullets in ParseGrounds); a successful write followed by a readiness fault exits 2 with no receipt, so a retry appends a duplicate entry; RecordGrounds enforces no bucket rule while the shipped-bucket exemption says never backfilled and the branch itself backfilled three shipped intents (refuse terminal buckets, or word the exemption honestly); missing versus malformed --grounds exit 2 versus 1 unevenly; the 36-record NOT READY consequence and the forward-only choice are recorded nowhere in the decision log.

## Grounds

- pursued: we expect a write whose result is visible on the plane the caller reads, and a gate whose reader enforces the floor it claims, to be what makes the argument trustworthy rather than merely present — an unreported write invites the duplicate that append-only recording cannot undo
