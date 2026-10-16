---
schema_version: 1
id: "iss-2610161947572570"
slug: "every-repository-abcd-manages-should"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "product thinker's request during the hosted-sites session"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/defaults/rules.json"
remedy: "Add a single-source-of-truth domain to the bundled rules every managed repository loads, and enforce it with a lint check that refuses a hand-edited generated block or a copy whose source has moved, by extending the existing generated-block drift checks rather than adding a new primitive."
---

Every repository abcd manages should carry an enforced single-source-of-truth rule: each fact has one canonical home, and every other place it appears is generated from that home and checked, never hand-copied. Today abcd holds pieces of this for itself (the Markdown-is-the-source rule in the bundled documentation domain, its own one-canonical-primitive principle, generated blocks with drift checks), but a managed repository gets no stated rule and no gate that catches a hand-edited copy drifting from its source.
