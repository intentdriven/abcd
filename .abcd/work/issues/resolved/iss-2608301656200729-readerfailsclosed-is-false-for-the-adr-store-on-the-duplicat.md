---
schema_version: 1
id: "iss-2608301656200729"
slug: "readerfailsclosed-is-false-for-the-adr-store-on-the-duplicat"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-189-round-5-ruthless"
found_at: "internal/core/lint/schema.go"
resolution: "The duplicate-key leg reads its own declaration, readerRefusesDuplicateKey, set on the issue store alone: capture's parseFrontmatterBlock refuses a duplicate by name, while the ADR dispatcher reads with frontmatter.Fields and renders an ADR carrying status twice on its first value with a nil error (probed through record.Describe). readerFailsClosed keeps the two legs that read a record's properties, where it is true of the ADR store. TestClosedSchemaAndDuplicateKeyClaimNoReaderWhereTheStoreHasNone now runs both halves of the split on the ADR store: the duplicate must not claim the refusal and the missing id must. The verification claim in iss-2608301519254418 is corrected to the unknown-key leg both reviewers actually checked, and the stale three-legs sentence in iss-2608301649337920's remedy with it."
impact: fix
resolved_by:
  intent: "itd-189"
---

readerFailsClosed is false for the adr store on the duplicate key leg and the record that resolved it claims all four stores were verified

Found by the round-5 ruthless review, and it lands on a record this branch
already resolved.

`readerFailsClosed` is set on the `adr` store and gates all three legs on the
godoc's assertion that "The three malformations differ; what the reader does
about them does not." That is false for `adr` on the DUPLICATE-KEY leg.
`record.readRecordHead` reads with `frontmatter.Fields`, the lenient scanner,
and no ADR reader anywhere refuses a duplicated key.

Probed: an ADR carrying `status: accepted` then `status: draft` renders --
`describeADR` returns the record with `Status:accepted` and a nil error --
while the gate says "the record reader refuses a duplicated key, so the file is
skipped by every ADR surface".

The part worth keeping: `iss-2608301519254418`, resolved on this branch in
round 4, asserts the flag is "correct for all four stores (verified
independently by both reviewers)". Both reviewers verified the UNKNOWN-KEY leg,
where `adr` has nil `knownFields` and returns early, and carried that silently
across to the duplicate-key leg, which does fire. A verification claim in a
resolved record is exactly the thing a later reader trusts instead of
re-deriving, so the record needs correcting alongside the code.

`TestClosedSchemaAndDuplicateKeyClaimNoReaderWhereTheStoreHasNone` covers
`adm`, `srp` and `iss`, and not `adr`.

Remedy: split the declaration so `adr` does not stand behind the duplicate-key
clause, and extend that test to the ADR store.
