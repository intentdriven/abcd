---
schema_version: 1
id: "iss-2609012259585969"
slug: "the-readings-companion-design-document-counts-five-dispositi"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "iteration-2 conformance audit against the design framework v4 and the readings companion v4, 2026-09-01"
origin: researcher-authored
production_mode: dictated-and-formatted
resolution: "The corrections are carried in the maintainer's amendment list for the two design documents, held outside the repository, with one of this record's own asks reversed the same day: the invocation stays position and target state, as section 4.1 says, and the scope operand this record asked the document to carry is withdrawn; the record's own account is the decision log's 2026-09-02 entries."
impact: internal
---

The readings companion design document counts five disposition states over a four-row table and still describes a two-operand invocation

The readings companion (v4) is the later of the two governing design documents
and the one the shipped code follows wherever the two differ. Two of its own
statements are wrong against its own tables and against the code.

The first is arithmetic. Section 4.3 announces a "Ratified vocabulary, five
states, with availability varying by position", and ruling R7 repeats "five
disposition states". The table under section 4.3 has four rows: `accepted`,
`rejected`, `declined` and `held`. The code ships those four in
`internal/core/issueschema/reading.go` and carries a comment recording the
five-versus-four discrepancy as knowingly unresolved; the v0.7.0 changelog was
corrected from "five" to "four" at the docs-currency gate. Nothing in the
companion names a fifth state, and the decision log's 2026-08-30 entry records
that the fifth stays unnamed. The count in the document is the defect.

The second is currency. Section 4.1 still states that "the operator supplies a
position and a target state" and that the invocation carries no free text on
that basis. adr-58 (2026-08-31) added a third operand, the scope, in a closed
grammar, and restated the binding property as "no operand carries prose". The
companion predates the ADR by two days and has not been amended.

Both are corrections to the document rather than to the code, and the document
lives outside the repository, so this record is the pointer: the next revision
of the companion should read "four states" and carry the scope operand in
section 4.1, and until it does a reader holding the document against the binary
will find the binary wrong where it is the document. The governing framework's
own pending amendments (the companion's section 12 plus adr-58 and ingest-time
minting) are held separately and are not this record's.

## Grounds

- deferred: the document lives outside the repository, so the record can only point at what it owes; the maintainer amends it
