---
schema_version: 1
id: "iss-246"
slug: "abcd-s-record-documents-a-surface"
severity: "major"
category: "drift"
source: "user-observation"
found_during: "intent-planning-interview"
found_at: ".abcd/development/brief/04-surfaces/README.md"
resolution: "sub-verb tables landed armed across 04-surfaces with the extended surface_coverage pass checking both directions against the command-tree snapshot; the Status enum stays two-valued, sub-verb rows carry the granularity"
impact: additive
resolved_by:
  intent: "itd-122"
  spec: "spc-27"
---

`surface_coverage` is blind inside a row, so `shipped` means "the top-level verb exists" and every unbuilt sub-verb hides behind it.

The `Status` column in `.abcd/development/brief/04-surfaces/README.md` is machine-checked: the `surface_coverage` record-lint rule asserts every `shipped` row has a backing `commands/<name>.md` or `skills/<name>/`, every `staged` row has none, and in reverse that every real surface has a row. That check is correct and it works — at surface granularity. It cannot see sub-verbs.

The consequence is visible in the registry itself: six of twenty rows read `shipped` and then qualify themselves in prose. `/abcd:embark` ("the full embark chapter ... remains a design target"), `/abcd:launch` ("packaging/publishing the artefact remains a design target"), `/abcd:intent` ("refine / grill / ship / consistency / shape / reclassify ... remain design targets"), `/abcd:capture` ("`promote` is a design target"), `/abcd` ("the richer board ... is a design target"), `/abcd:banlist` ("the remaining slices"). The partial-ness is honestly recorded and entirely unenforceable, because it lives in prose the lint does not read.

Documents outside the registry then cite those sub-verbs as live, because nothing checks them. Three did so at the time of filing, all now corrected: `02-constraints/04-naming.md` called `capture promote` "live as of spc-30/itd-46"; `05-internals/01-agents.md` said "spc-29 ships" both `intent consistency` and `intent shape`; the `intents/README.md` lifecycle diagram presented `intent ship`, `intent reclassify`, an `intent_lifecycle_hook`, and the bundle and discipline paths of `intent plan` as live steps. Each cites a **predecessor-store** spec id (spc-29, spc-30): the capability shipped in an older spec store and the prose survived the migration to the native one.

Two defects, one fix. The `Status` enum is two-valued while reality is three-valued (fully / partly / not at all), and the check does not reach sub-verb granularity. Per adr-40 both close together: each surface file under `04-surfaces/` carries a sub-verb table recording two facts per verb — which bucket it is (lint / review / audit / gate) and whether it exists (`shipped` / `staged`) — and `surface_coverage` extends to check each row against registered cobra sub-commands in both directions. A `staged` row for `capture promote` then cannot be called live elsewhere without failing the build.

Supersedes the original text of this entry, which claimed abcd's record documents a surface substantially larger than the binary with no gate catching the drift. That framing was wrong: the marking convention exists (some hundred inline `design target, not yet shipped` markers), the detector exists (`surface_coverage`, iss-35), and the brief is aspirational by design per adr-5. The defect is narrower and precisely located.
