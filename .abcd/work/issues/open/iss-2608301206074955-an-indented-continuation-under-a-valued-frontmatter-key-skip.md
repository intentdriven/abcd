---
schema_version: 1
id: "iss-2608301206074955"
slug: "an-indented-continuation-under-a-valued-frontmatter-key-skip"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-179-round-2-security"
found_at: "internal/core/lint"
---

an indented continuation under a valued frontmatter key skips the record in every reader while record_schema reports nothing

Found by the round-2 adversarial security review of build/itd-179, and
recorded rather than chased: PRE-EXISTING and not grounds-specific.

A record carrying a key with a value on its own line followed by an indented
continuation line makes `parseFrontmatterBlock` return "unexpected indented
line", so every capture surface SKIPS the record, while `record_schema` reports
zero findings -- the block look-ahead only fires when the same-line value is
empty. Reader and gate therefore disagree, in the direction that hides a
record.

The reviewer reproduced the identical verdict pair (reader refuses, gate finds
nothing) with the same continuation under `slug` and under `impact`, so it
belongs to the standing "gate and reader disagree" family rather than to this
branch. Siblings already captured: iss-2608300205044566 (single-quoted scalars
split the gates) and iss-2608300234598982.

A differential harness over 41 `grounds:` spellings found this to be the ONLY
divergence: the round-1 fix 050f3366 closed the parity class for grounds
itself.
