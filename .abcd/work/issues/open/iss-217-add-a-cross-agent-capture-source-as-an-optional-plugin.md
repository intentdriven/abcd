---
schema_version: 1
id: "iss-217"
slug: "add-a-cross-agent-capture-source-as-an-optional-plugin"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
details: "SOTA-researched 2026-07-30, decision recorded (DECISIONS.md 2026-07-30): a cross-agent capture source is the current default SOTA upgrade once cross-harness compatibility becomes a concern, import-only over the native store via the pre-declared specstory-import seam. Candidate tool, target harness, and provider caveats are named in .abcd/development/research/notes/2026-07-30-session-recording-sota.md"
suggested_fix: "Build the specified-but-unbuilt import path: parse the capture tool's per-session markdown and merge by timestamp/content hash through history.Capture(kind=specstory-import), so the two-stage fail-closed redaction applies to imported material; telemetry opt-out in any wiring, cloud/sharing tiers out of scope, the tool's in-repo history directory never tracked."
related_issues: ["iss-95", "iss-96", "iss-125", "iss-157"]
remedy: "Waits on ruling K (keep parked or close): if kept, name the trigger in this record (the first session a second harness produces) and, when it fires, build the import path in `internal/core/history` that parses the tool's per-session markdown and merges by timestamp and content hash through `history.Capture` on the pre-declared import kind, labelling tool and route separately per ruling J13, proven by a test that an imported transcript with a planted secret and home path lands redacted and a second import adds nothing; if closed, wontfix it with that reason and leave the pre-declared seam for a later intent."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed K): Keep the cross-agent capture source parked until a second harness matters, or close it now?"
---

add a cross-agent session-capture source as an optional plugin

SOTA survey (2026-07-30, `.abcd/development/research/notes/2026-07-30-session-recording-sota.md` — candidates are named there):

- What an external capture source would add over the native store (adr-29,
  itd-89): capture of sessions from non-Claude harnesses, normalised to one
  per-session markdown format, and a reader for the legacy pre-rebuild
  transcript archives (the Phase-0 corpus of 439 transcripts is an immediate
  import candidate).
- Importing through the seam keeps the native store's posture for imported
  material: storage stays out of the repo, provenance hashing applies, and the
  two-stage fail-closed redaction (secrets, paths, identity) runs on import
  exactly as on native capture. The `SessionEnd` hook remains the capture path
  for Claude Code sessions.
- The record pre-declares the seam: `--kind specstory-import` exists, the
  verification matrix promises timestamp/content-hash merge over the same
  store, but no merge/parse logic is implemented — that is the plug-in.
- Under sota-per-intent this is path 2 (native floor + seam) being exercised,
  not a new required dependency; guardrails are the existing invariants
  applied at the import boundary (one door through `history.Capture`,
  imported transcripts treated as untrusted, records and docs name no tool
  before adoption is decided).
- Related gaps that shaped import quality at research time: iss-95 (capture
  precondition) and iss-96 (unanchored secrets) remain open; iss-125
  (hostnames) and iss-157 (network identifiers) have since been resolved by
  the scanner's network-pattern work.

Decision (2026-07-30, DECISIONS.md): a cross-agent capture source is the
**current default SOTA upgrade**, triggered when compatibility beyond Claude
Code becomes a concern. Adoption shape is import-only over the native store;
the native floor stays the default and capture degrades to it when the tool is
absent. The candidate tool, the target harness, and the provider-coverage
caveats binding at adoption time are recorded in the research note's addendum.
The SOTA verdict is re-verified at the adopting intent, and a move from
optional to required tool is a sota-per-intent path-1 hard stop.

(Originally captured 2026-07-30 as iss-161 against a stale ledger; re-minted
as iss-217 after the id collision with main's iss-161.)

## Remedy grounds (2026-09-29)

- The remedy keeps the suggested_fix (kept above) and extends it with the J13 labelling ruling and a named trigger, so the parked record has a condition the ledger can check.
- SOTA check: the candidate tool's upstream README (https://github.com/specstoryai/getspecstory, read 2026-09-29) lists eleven terminal agent harnesses, writes history inside the project directory and is Apache-2.0, so the research note's adoption shape (import-only, the in-repo history never tracked, telemetry opted out) still fits; the README does not state the telemetry default, so the note's reading stands until re-verified at adoption.
- Rejected: a second store or a capture path that bypasses `history.Capture` (the one-door rule).
