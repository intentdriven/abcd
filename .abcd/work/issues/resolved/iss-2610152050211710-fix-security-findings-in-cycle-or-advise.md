---
schema_version: 1
id: "iss-2610152050211710"
slug: "fix-security-findings-in-cycle-or-advise"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "PT ruling 2026-10-15 ~22:40 BST"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/rules.json"
remedy: "Add the rule to the SECURITY domain in .abcd/rules.json, so every session and agent that works on a security finding is taught it: fix a security-relevant finding found during development or review within the same development cycle; where that is not possible, record it as a private advisory and never name it in a commit message, pull-request body, issue, comment or public record."
resolution: "The rule is now the fourth SECURITY-domain rule in .abcd/rules.json, injected by the rules loader into every session working on a security finding."
impact: internal
resolved_by:
  commit: "3ba387595"
---

The product thinker ruled on 2026-10-15 that a security-relevant finding identified during development or review is fixed within the same development cycle, and that where this is not possible it is recorded as a private advisory and mentioned nowhere in commits, pull requests or other public text. Today a security finding a review raises is handled like any other: fixed, or captured as an open issue and named in the commit and PR trailers, which publishes a weakness before its fix ships. The rule is taught nowhere an agent reads before it acts.
