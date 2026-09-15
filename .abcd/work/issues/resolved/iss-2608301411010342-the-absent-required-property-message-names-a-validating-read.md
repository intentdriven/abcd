---
schema_version: 1
id: "iss-2608301411010342"
slug: "the-absent-required-property-message-names-a-validating-read"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "itd-189-round-3-ruthless"
found_at: "internal/core/lint/schema.go"
resolution: "The missing-property message's reader clause is now declared per store (readerFailsClosed), true for the issue ledger and the ADR store whose readers refuse an incomplete record and skip it, and absent for the admission and surprise stores which have no such reader. What the admission reader actually does with an incomplete record is pinned by a test rather than assumed, and the absent-versus-blank test moves onto an issue fixture, where the claim it asserts is true."
impact: fix
resolved_by:
  intent: "itd-189"
  spec: "spc-67"
---

the absent-required-property message names a validating reader for two stores that have none, and this branch certified that claim as store-wide

Found by the round-3 adversarial ruthless review of build/itd-189.
The MESSAGE is pre-existing; a57e7bb6 newly certified it, in a comment and in a
test assertion, as holding "for every store and every field".

`checkRecordRequiredFields`' absent branch says the store's reader "validates
before it reads, so a record without it is skipped — invisible to every <noun>
surface while it still sits in the store". That holds for the two stores whose
readers fail closed: the issue ledger (capture's validateStrict) and the ADR
store (the record dispatcher confirms the frontmatter id). It does not hold for
the two stores this cycle added.

The only reader of admission records anywhere in the tree is
`admittedProposals` in internal/core/lint/readingoutstanding.go, and it
validates nothing but a non-empty `proposal` and the run agreement: an admission
carrying only `run` and `proposal` — no schema_version, no id, no grounds — is
read and honoured, and silences its proposal in the outstanding report. The
record is not skipped and not invisible; it is actively counted. For surprises
there is no reader at all, so the message names one that does not exist.

This is the class iss-2608301308369559 closed for the BLANK branch, left open in
the ABSENT branch one line above it: what a reader does with a record is a
property of the reader, and a store with no validating reader supports no claim
about one.

Remedy: declare on `recordStore` whether the store's reader validates before it
reads, true for the issue and ADR stores and false for the admission and
surprise stores, and drop the reader clause where it is false. The absent half
of TestAbsentAndEmptyRequiredPropertiesGiveDifferentReasons is pinned on an
admission fixture and belongs on an issue one, where the claim it asserts is
true.
