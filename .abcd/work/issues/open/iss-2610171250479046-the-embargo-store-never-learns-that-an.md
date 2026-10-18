---
schema_version: 1
id: "iss-2610171250479046"
slug: "the-embargo-store-never-learns-that-an"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "publishing an advisory under the product thinker's standing rule, 2026-10-17"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/embargo"
remedy: "Have intake (and every plan) read each held advisory's current state on GitHub and record published_at or closed_at when GitHub reports it, releasing it from the check, as commands/embargo.md already promises; pin it with a test where the forge reports a held advisory published."
---

The embargo store never learns that an advisory was published outside abcd embargo publish: after a draft advisory was published on GitHub on 2026-10-17, abcd embargo intake and a fresh publish plan both left the store's state at draft, and abcd embargo check still refuses the advisory's id in text, so the record cannot be restored to the public ledger with the gates passing. commands/embargo.md says a plan records published_at for an advisory GitHub already reports published.
