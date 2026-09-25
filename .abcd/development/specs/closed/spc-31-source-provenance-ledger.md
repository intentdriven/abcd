---
id: spc-31
slug: source-provenance-ledger
intent: itd-76
---
# source-provenance-ledger

## Summary

Delivers the personal core of the sources corpus as native abcd verbs:
`abcd source add`, `abcd source ledger`, `abcd source sync-banlist`, and
`abcd source cite-check`, operating on the user-level home
(`~/.abcd/sources/` by default; relocation is itd-77's concern). Team
share/ingest and paper reconstruction are out of scope here — they are
itd-126 and itd-127, both `refines` itd-76. The trust boundary the verbs
enforce is adr-41 / brief invariant 9, cited, not restated.

## Scope

- **Store layout** (per the 2026-07-08 scaffold plan and SOTA note):
  `sources.json` — a CSL-JSON array whose `custom` block carries
  `confidential`, `permission_status`, `keywords`, `aliases`, `ban_authors`;
  per-source folders `confidential/<key>/` and `public/<key>/` holding the
  original, extracted text, and derived artefacts — **folder location is the
  classification**; `ledger/<repo>.jsonl` — one append-only file per
  consuming repo. The corpus directory is itself a no-remote git repository;
  its history is the tamper-evidence layer (itd-16 is a possible later
  backend; nothing depends on it).
- **Verbs** (transport-agnostic core + CLI/plugin front doors, per the
  wired-or-it-isn't-done rule):
  - `add` — registers a document: writes the CSL-JSON entry, stores the
    files under the class folder, commits the corpus repo. Declassification
    is `git mv confidential/<key> public/<key>` plus the entry update — a
    visible move, never a silent flag edit.
  - `ledger` — appends `{ts, repo, decision_ref, claim, source_key, locator,
    influence, cited_publicly}` and commits; corrections are new lines. The
    `cited_publicly: true` flip is a distinct invocation that checks the
    source's `permission_status` first and refuses, naming the failing gate,
    when permission is absent (adr-41 gate 2 exercising gate 1).
  - `sync-banlist` — projects confidential entries' titles and aliases
    (authors only under `ban_authors: true`) into the repo's untracked
    itd-74 private banlist block, between generated-block markers it owns.
  - `cite-check` — scans stdin or a file; reports offending entries **by key
    only** so its own output is shareable.
- **Guard wiring**: the pre-commit guard calls `sync-banlist --refresh`
  when a corpus is present; when absent, every corpus-dependent step prints
  a one-line "corpus absent — skipped" notice (loud staging) and exits 0.

## Approach

Go core package (`internal/core/source`) owning store I/O, classification
moves, ledger append, and the two-gate check; front doors stay thin. All
matching for banlist and cite-check reuses the itd-74 machinery rather than
growing a second matcher (one canonical primitive) — note the recorded
RE2/ASCII word-boundary hazard there: matching must use Unicode-aware
predicates, not `\b`. No network anywhere (brief invariant 7); no new
dependencies — CSL-JSON parses with encoding/json.

## Acceptance criteria → design

1. *Add & classify* — `add` writes entry + files under the declared class
   folder; the folder path is derived from, and only from, the declared
   class at ingestion.
2. *Ledger append-only* — `ledger` opens the per-repo JSONL in append mode;
   there is no edit path; the corpus commit history proves it.
3. *Banlist projection* — `sync-banlist` regenerates its owned block; the
   itd-74 pre-commit guard, already shipped, refuses banned strings; the
   refresh hook makes the block at most one commit stale.
4. *Safe scan* — `cite-check` output contains keys and byte offsets only,
   never the matched strings themselves.
5. *Two-gate flip* — the flip subcommand hard-fails on `permission_status`
   ≠ permitted, and a successful flip is itself an appended line.
6. *Loud absence* — every verb and the guard hook detect a missing corpus
   and say so on one line, exit 0 (guard) / a distinct no-corpus exit (verbs).
7. *Visible declassification* — the move updates class by location; the next
   `sync-banlist` run drops the moved key's strings; flip eligibility follows
   `permission_status`, which declassification sets.

## First validation

Re-establish this repository's own corpus through the shipped verbs (the
dogfood-target section of the intent): ingest the acknowledgements baseline's
public sources, wire the guard, and run one full consult→ledger→flip cycle.
