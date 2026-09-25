---
schema_version: 1
id: "iss-2608230752354928"
slug: "the-history-store-s-validkinds-drops-the-source-harness-of-a"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "record-review"
found_at: "internal/core/history/store.go"
details: "history.Capture validates source kind against validKinds = {native, specstory-import} (internal/core/history/store.go). A transcript from any other harness can only be stored by declaring it 'native', so the record asserts abcd's own hook produced it and the true source is lost. source_kind is the only provenance channel on the record; nothing else on the frontmatter names a harness."
suggested_fix: "Separate the ingest route from the source harness before the cross-agent import path is built. Either widen the kind vocabulary, or add a distinct source-harness field and let kind describe only the route. Note that specstory-import already conflates the two, so the seam pre-declared in the brief inherits the same defect."
related_issues: ["iss-217"]
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Is source_kind keyed by harness or by ingest route, decided before second-harness capture ships?"
---

the history store's validKinds drops the source harness of an imported transcript

`validKinds` accepts exactly `native` and `specstory-import`. Capture refuses
anything else, so a transcript from another harness reaches the store only by
being declared `native` — a positive claim that abcd's own SessionEnd hook
produced it.

`source_kind` is the sole provenance channel on a record. The frontmatter
carries `session_id`, `root_commit`, `captured_at`, `source_kind`,
`source_sha256`, and the two redaction counters; none of the others names a
harness. So the misdeclaration is unrecoverable from the record.

This matters now rather than later because the store is otherwise already
harness-agnostic. `Capture` treats the transcript as opaque bytes, with no
JSONL parsing and no message-shape assumption, and the key is the git
root-commit SHA rather than anything about the host. The kind field is the one
place harness coupling survives, and it is the field that would carry the
answer.

Adjacent, and the reason to settle the shape first: `specstory-import` already
names a tool where the sibling value names a route, so the vocabulary conflates
ingest path with source. iss-217 plans the cross-agent import over exactly this
seam, so the shape is better decided before that work than during it.

Additional evidence (2026-08-26, second-harness adaptor lab, local tier): the
lab's capture path exercises this defect live. Transcripts exported from a
non-native host reach the store through `hook session-end` and are recorded as
`source_kind: "native"` — the exact misdeclaration this issue predicts, now
demonstrated rather than hypothetical. The rest of the pipeline held: the
store treated the foreign transcripts as opaque bytes and redacted them end to
end (twelve sessions stored in one drain), confirming the kind field is the
sole remaining harness coupling and must be settled before any second-harness
capture ships.
