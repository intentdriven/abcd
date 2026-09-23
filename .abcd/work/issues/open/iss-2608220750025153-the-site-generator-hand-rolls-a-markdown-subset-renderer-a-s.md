---
schema_version: 1
id: "iss-2608220750025153"
slug: "the-site-generator-hand-rolls-a-markdown-subset-renderer-a-s"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "agent-observation"
found_at: "internal/core/site"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: dependency decision on goldmark and x/net/html for the site generator). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

the site generator hand-rolls a Markdown-subset renderer, a strict HTML tokenizer and a CSL formatter to stay dependency-free; adopting goldmark and x/net/html instead is a maintainer dependency decision