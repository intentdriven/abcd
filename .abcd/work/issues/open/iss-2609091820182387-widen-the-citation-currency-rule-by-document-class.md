---
schema_version: 1
id: "iss-2609091820182387"
slug: "widen-the-citation-currency-rule-by-document-class"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "release-gate"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/contextcurrency.go"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: widen citation-currency by document class (brief, principles, draft/planned intent decision refs) with a supersedes exemption). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

The citation-currency rule already resolves a record handle and refuses a citation whose target is terminal, reading an ADR's status unioned with a declared supersession so that a half-declared supersession still counts as out of force. It deliberately reads one section of one document, and its own header gives the reason: a terminal-state citation is normal everywhere else, because a shipped intent's evidence trail, a supersession chain and a dated plan all name closed records legitimately, and only the living orientation document claims to describe what is true right now. A hand sweep after two records were superseded gives that judgement its first measurement. Of roughly forty-five citations of superseded decision records across the tree, eleven were wrong and the rest were correct provenance, so a rule applied everywhere would have been three-quarters false alarm, and the discriminator is what the citing sentence is doing rather than anything resolvable from the link. The tractable middle is to widen the same rule by document class rather than by sentence: extend its target from one file to the documents that assert the present tense by charter, which are the brief chapters, the principles, and the decision-record references carried in the frontmatter of an intent still in drafts or planned. That set is closed and declarable, it keeps the fail-closed shape the rule already has, and measured against the same sweep it would have caught ten of the eleven. The eleventh is a comment in a test file, which is out of reach for the same reason no gate reads decision prose against code identifiers. Two false alarms would survive even the narrowed form, both of the shape where a handle appears beside the word supersedes, which suggests exempting a handle within a few words of that token. Detector: a brief chapter, a principle, or a planned or draft intent's decision-record frontmatter citing a superseded record as current authority is refused, while a supersession chain, a historical plan and a resolved record naming what they found stay clean.
