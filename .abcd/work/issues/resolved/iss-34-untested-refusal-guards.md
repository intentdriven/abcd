---
schema_version: 1
id: "iss-34"
slug: "untested-refusal-guards"
severity: "major"
category: "tech-debt"
source: "agent-finding"
found_during: "2026-07-08 multi-agent review"
found_at: "internal/core/launch/bundle.go"
resolution: "Acceptance corpus complete: detectors added for the symlink dereference accept path and its cycle guard, the scripts runtime-closure guard (all three outcomes plus a no-closure polarity control), the MQ001 quotation budget (both refusals plus an under-budget control), the multi-source half of the ML001 licence check, and the ask --file-back write path (accept path plus three refusals, each proved side-effect-free against a byte-level store snapshot). The ML001 single-source branch was already covered by TestLintRejectsMalformed and needed nothing. Every case was watched fail with its guard disabled; no guard was found broken, so the increment is test-only."
impact: internal
---

refusal guards with zero coverage: the launch bundle symlink-dereference and scripts-deny guards (internal/core/launch/bundle.go:339), the memory quotation-budget and licence-detection compliance checks (internal/core/memory/lint.go:225), and the memory ask --file-back write path (internal/core/memory/ask.go:354) are all untested. A guard fails silent: when it regresses the system keeps working and simply stops refusing. Detector (per guards-prove-themselves): a convention that every refusal path ships a test presenting the forbidden input and asserting the rejection, its error shape, and the absence of side effects; a pairing lint between declared invariants and named tests is the promotion path. Acceptance corpus: the five guard paths above.

---

**Closed (2026-08-02, v0.5.0 item C8):** the five-path acceptance corpus is
complete. The audit that opened the round found the entry's "zero coverage"
claim accurate for three paths, half-accurate for one, and already satisfied for
one:

- **Symlink dereference** (`bundle.go` `handleSymlink`/`walkSymlinkTarget`) —
  PARTIAL. The reject half was already pinned by `TestSymlinkEscapeRejected` and
  `TestSymlinkToRepoRootDoesNotLeakDenied`. What was missing is the half a guard
  whose job is to dereference SAFELY also owes:
  `TestSymlinkDereferenceAcceptsInRepoTarget` pins that an accepted link to an
  in-repo file is included under its own logical path with the TARGET as
  `ResolvedPath`, that a link to an in-repo directory emits its subtree under the
  link's prefix, and that the target tree does not additionally ship under its
  own unrequested name; `TestSymlinkDirectoryCycleRejected` pins the
  `ancestors` cycle guard, whose absence does not merely mislabel a rejection —
  the resolve never terminates.
- **Scripts runtime closure** (`classifyRegular`) — UNCOVERED, now
  `TestScriptsNotInRuntimeClosureRejected`, which walks all three outcomes in one
  resolution through the injectable `ClosureFn` seam (literal include outside the
  closure -> `rejected(fs_error, scripts_not_in_runtime_closure)`; swept up by a
  directory include -> `excluded(denied_namespace)`; matching the closure's own
  `__pycache__`/`.pyc` deny -> likewise excluded, not rejected) and adds a
  no-closure polarity control so the exclusions are demonstrably the gate's doing.
- **MQ001 quotation budget** (`lint.go` `checkQuotation`) — UNCOVERED, now
  `TestLintQuotationBudgetMQ001`: both refusals (the contiguous quoted-span
  ceiling and the per-page share of a source), an under-budget control, and the
  two properties around the finding — MQ001 is curator-advisory (warn, exit 0),
  and lint leaves the page byte-identical.
- **ML001 licence detection** (`lint.go` `checkLicence`) — the single-source
  branch was ALREADY covered by `TestLintRejectsMalformed/external_missing_licence`,
  which presents the forbidden page and asserts the blocker, its code and the
  exit contract; that half was left alone rather than padded. The multi-source
  branch (an external `sources[]` entry with no `licence` key beside a sibling
  that has one) had no case and gains one in the same table.
- **`ask --file-back` write path** (`ask.go` `fileBack`/`fileBackSource`) —
  UNCOVERED, now `TestAskFileBackWritesCitedPage` (a page filed back without its
  own source block inherits the cited matches' provenance, validates, cites the
  ingested hash, and is back-linked in the registry) and
  `TestAskFileBackRefusesWithoutWriting` (no cited matches, a declining decision,
  and cited pages without complete provenance — each asserting the typed
  `*AskError` or the declined status, the specific message, and, against a
  byte-level snapshot of the whole store taken before and after, that nothing was
  written, mutated or removed).

Detector-first throughout: each new case was watched fail with its guard
disabled and pass with it restored — the scripts-closure branch short-circuited,
the cycle check disabled, the symlink file dereference dropped, `checkQuotation`
and `checkLicence` stubbed out, and the ask-side no-match refusal, decline gate,
provenance requirement and write call each neutralised in turn. No guard was
found broken and no production behaviour diverged from its own documentation, so
the increment is test-only and carries no CHANGELOG entry — the repo's
definition of done ties an entry to a user-facing change, and coverage alone
is not one. `go.mod` is untouched.

The entry's promotion path — a pairing lint between declared invariants and
named tests — is NOT built here; it remains available as a separate convention
increment, since the corpus this entry scoped is coverage, not the lint.
