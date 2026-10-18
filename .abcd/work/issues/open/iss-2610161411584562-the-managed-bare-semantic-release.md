---
schema_version: 1
id: "iss-2610161411584562"
slug: "the-managed-bare-semantic-release"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "security review of the release-chain fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal"
remedy: "Give the bare+semantic profile the same pushed-tag handling as abcd's own release workflow, or drop its tag-push trigger, with a scaffold parity test."
---

The managed bare+semantic release profile keeps the tag-push trigger but has no pushed-tag job, so a hand-pushed tag signs the receipts and then fails its own pin to the default branch ref.
