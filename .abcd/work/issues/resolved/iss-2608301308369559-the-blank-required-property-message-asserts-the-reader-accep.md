---
schema_version: 1
id: "iss-2608301308369559"
slug: "the-blank-required-property-message-asserts-the-reader-accep"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-189-round-2-security"
found_at: "internal/core/lint/schema.go"
resolution: "The generic blank-required-property finding no longer states what the reader does with a blank, because that is a per-field property: capture refuses thirty-nine of the forty-two required-issue-field x blank-spelling combinations and accepts three, every one of them a found_during. The legs that mirror capture's own shape checks (the enum legs, the slug grammar) now judge a present-but-blank scalar and state the refusal on their own terms, and the generic leg stays silent for a field one of them has already answered. Every combination is pinned against the reader itself by TestBlankRequiredPropertyFindingsMatchTheReadersVerdict."
impact: fix
resolved_by:
  intent: "itd-189"
  spec: "spc-67"
---

the blank-required-property message asserts the reader accepts a value it actually refuses inverting the defect the previous commit fixed

Found by the round-2 adversarial security review of build/itd-189.
INTRODUCED BY THIS BRANCH, and it is the branch's own last commit's defect
class INVERTED: 8f20a9c7 fixed a message that claimed the reader SKIPS a record
it accepts; this one claims the reader ACCEPTS a record it skips.

For `severity: ""` HEAD emits: "required property 'severity' carries no value
once its YAML scalar is read; ... the issue reader type-checks a present
property without judging it, so this record is read and every surface renders
the property as answered."

`validateStrict` in fact refuses and skips. The reviewer measured every
required issue field against `""`, `''` and `" "`: **20 of 21 combinations are
refusals** (`invalid severity ""`, `slug "" is not kebab-case`, `id "" does not
match ^iss-[0-9]+$`, `found_during must be non-empty`, `unsupported
schema_version`); only `found_during: ''` is accepted.

The cause is this branch's own widening: `isAbsentValue` now short-circuits
`checkIssueRecordShape`'s enum leg at schema.go:856 before it can speak. The
base emitted the ACCURATE message (`invalid severity ''; capture refuses a
value outside {...} and skips the record`).

No security consequence -- the record is still a blocker either way, so nothing
leaks and nothing merges that should not. It is an accuracy defect, and it is
reported because this repo's stated standard is that the rule never makes a
confident false statement. A gate that misdescribes why it refused sends the
operator to the wrong remedy.

Remedy (class-level): the "reader accepts a blank" clause is a PER-FIELD
property, not a per-store one. Either let `checkIssueRecordShape`'s enum, slug
and id legs fire on a present-but-blank value (dropping the isAbsentValue
short-circuit for the fields the reader judges), or carry a per-store statement
of what its reader does with a blank and read the message from that.
