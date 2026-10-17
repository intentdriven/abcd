---
schema_version: 1
id: "iss-2610171250466572"
slug: "abcd-embargo-publish-can-never-publish"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "publishing an advisory under the product thinker's standing rule, 2026-10-17"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/embargo"
remedy: "Let the plan take the fix itself as the evidence: an advisory triaged as a vulnerability whose fix commit (recorded at triage or resolve time in the private store) is reachable from a published release tag is planned with the first such tag as its patched version, with no public record required; after publication the verb prints the command that restores the record as resolved. Pin it with a test where the record is only in the store and the fix is in a tag."
---

abcd embargo publish can never publish an advisory handled the way the security rule asks: its plan includes an advisory only when the advisory's record is resolved in the release, but the security rule keeps that record out of every committed tree until the advisory is published. On 2026-10-17 the plan for v0.15.0 left out an advisory whose fix had shipped in v0.15.0 with the reason that its record is not resolved there, and it was published by hand through the forge API instead, as the release-chain advisory was before it.
