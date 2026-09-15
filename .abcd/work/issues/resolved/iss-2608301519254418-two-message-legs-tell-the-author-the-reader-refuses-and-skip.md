---
schema_version: 1
id: "iss-2608301519254418"
slug: "two-message-legs-tell-the-author-the-reader-refuses-and-skip"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-189-round-3-ruthless"
found_at: "internal/core/lint/schema.go"
resolution: "The closed-schema finding gates its reader clause on readerFailsClosed, the same declaration the missing-property finding consults, so both legs that read a record's PROPERTIES rest on one statement about the store rather than one leg declaring and the other assuming. The duplicate-key finding gates on a declaration of its own. The issue store keeps the refusal account on both legs and the ADR store keeps it on the closed-schema leg, which is true of them; the admission and surprise stores state what a key outside a closed schema and a duplicated key ARE, and stop there. The duplicate-key leg is a THIRD reader question and does not stand behind this flag: the ADR dispatcher reads with the lenient scanner and refuses no duplicate, so it is declared separately as readerRefusesDuplicateKey (iss-2608301656200729)."
impact: fix
resolved_by:
  intent: "itd-189"
  spec: "spc-67"
---

two message legs tell the author the reader refuses and skips their record when the admission reader counts it and the surprise store has no reader at all

Found by the round-3 ruthless review. FIX-FIRST. **The fourth instance on this
branch of one class: a message asserting a mechanism that is not the case.**

`checkRecordUnknownFields` (schema.go:786-788) emits, as a BLOCKER, that a key
outside the schema "makes this a record the reader refuses and skips —
invisible to every admission surface while it still sits in the store". Probed:
`ReadReadingOutstanding` returns `Unadmitted=[]` — the record is not skipped and
not invisible, it is ACTIVELY COUNTED. For `srp` the message is worse: no
surprise reader exists anywhere in the tree, so "invisible to every surprise
surface" names an empty surface set and a refusal nobody performs. The author is
sent to look for a record that is being read.

`checkRecordDuplicateKeys` (schema.go:1201-1203) carries the same claim for
EVERY store: "the record reader refuses a duplicated key, so the file is skipped
by every admission surface". True for `iss` (capture/parse.go:122); false for
`adm`, `srp`, and — pre-existing — `rdi`, `rdg`, `dsp`.

The sting: `6226f7d2` fixed exactly this defect in `checkRecordRequiredFields`
one function away, and did not check the two legs beside it. The flag it added,
`readerFailsClosed`, is correct for all four stores on the UNKNOWN-KEY leg,
which is the leg both reviewers checked: `adr` declares no `knownFields` and
returns early, so the claim is never rendered there.

It is NOT correct for `adr` on the DUPLICATE-KEY leg, and the reviewers carried
their unknown-key verdict silently across to it. `record.readRecordHead` reads
with `frontmatter.Fields`, the lenient scanner, and no ADR reader anywhere
refuses a duplicated key: an ADR carrying `status` twice renders, with the first
value and a nil error. The duplicate-key leg is declared separately, as
`readerRefusesDuplicateKey` (iss-2608301656200729).

Remedy, one condition, both sites: gate the reader clause on
`r.store.readerFailsClosed` exactly as `checkRecordRequiredFields:745` now does,
keeping the schema-closed half unconditional. `adr` is unaffected on the
closed-schema leg (nil knownFields, early return) and must not keep the claim on
the duplicate-key leg; `iss` keeps both, which is true of it. Then widen the
flag's godoc: it covers required properties and the closed schema — one
declaration, two legs, rather than one leg declaring and one assuming — with the
duplicate key on its own declaration beside it.
