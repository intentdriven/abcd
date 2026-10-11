---
schema_version: 1
id: "iss-245"
slug: "the-capture-store-s-schema-models"
severity: "major"
category: "architectural-insight"
source: "user-observation"
found_during: "intent-planning-interview"
found_at: "internal/core/capture/capture.go:72,92"
resolution: "Both halves shipped. capture resolve takes --intent/--spec/--commit and writes the resolved_by object (81c5b474); capture promote mints an intent draft and stamps promoted_to (bf462e47), and 3 issues now carry it. The observation that remains — the verbs exist but nothing compels or verifies their use — is carried forward as iss-2608241347321757."
impact: internal
resolved_by:
  commit: "81c5b474"
---

The capture store's schema models resolution provenance it has no verb to write. ResolvedBy{Intent,Spec,Commit} (capture.go:72) is parsed on read (validate.go:275) but Resolve (workflow.go:162) writes only resolution and impact and exposes no flag to supply it, so a resolved issue can assert it was fixed in prose but cannot point at the intent, spec or commit that fixed it. PromotedTo (capture.go:92) is validated against ^itd-[0-9]+$ (validate.go:121) and documented in the issues README as 'the itd-N this issue graduated into', yet no verb writes it and no issue in the ledger carries it — so the capture-to-intent promotion the schema anticipates is unreachable, which is why the 'Which ledger?' note in commands/intent.md forces the intent-vs-issue call at the moment of lowest information with no way to revise it. Contrast the intent side, where the fidelity review binds its verdict to a sha256 receipt over the acceptance criteria: one record store, two very different evidence standards.