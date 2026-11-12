---
schema_version: 1
id: "iss-2610251558360735"
slug: "docs-explanation-rationale-md-3-says"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D2"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/explanation/rationale.md"
remedy: "Align rationale.md with the three roles, or say that the two people work with the agent team as the third role."
resolution: "docs/explanation/rationale.md now says abcd's bet is a small team of three roles (the product thinker, the facilitator and the team of AI agents), as roles.md and adr-2609151528057260 do; the figure's alt text no longer counts two roles."
impact: fix
resolved_by:
  commit: "ed24dbd770a5c5c788752ef186d67d6828268ffd"
---

docs/explanation/rationale.md:3 says abcd's bet is a small team of two roles; docs/explanation/roles.md:3 says abcd is built for three roles, per ADR-2609151528057260, whose Context retires the two-person-team description.

## Grounds

- pursued: rationale.md and roles.md give the same count of roles, three, and name the same three; a 'two roles', 'two-person' or 'two people' phrase in rationale.md, or the page not naming the team of AI agents, would show it wrong (detector.sh in the lane directory checks this; it failed before the commit and passes after).
