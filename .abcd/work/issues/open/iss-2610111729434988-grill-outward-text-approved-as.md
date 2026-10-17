---
schema_version: 1
id: "iss-2610111729434988"
slug: "grill-outward-text-approved-as"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "security triage and fix run, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/defaults/rules.json"
remedy: "Add a GRILL rule: a question that approves text to be posted quotes the exact text, even above the addressee's knowledge floor, with a one-line plain summary beside it; prove the generated rule text and the question check's acceptance of such a question with tests"
---

When an abcd interview asks the product thinker to approve text that will be posted outside the repository (an advisory, a pull-request body, an issue), the register rule rewrites that text into product terms, so the person approves a paraphrase and the agent then posts a different, technical wording. On 2026-10-11 a product-register question approved a plain-words sentence for a draft security advisory, and the sentence actually posted was a technical rewording the person never saw. The GRILL asking rules (bundled rules domain) say to quote the thing being decided, but the register rule overrides it for this case.
