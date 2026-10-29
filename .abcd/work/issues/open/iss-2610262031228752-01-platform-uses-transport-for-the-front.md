---
schema_version: 1
id: "iss-2610262031228752"
slug: "01-platform-uses-transport-for-the-front"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 120 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/01-platform.md:17"
remedy: "Edit glossary/core/transport.md: record the front-door sense used in 'transport-agnostic core' and in the surface entry's 'transport package', or point that sense at surface, so the term is not left with only the oracle sense."
finding_ref: cfk-39f510dbd783e81a
---

01-platform uses transport for the front doors of a transport-agnostic core; the glossary defines transport only as the mechanism delivering context to an oracle.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/01-platform.md:17` — "The core is a transport-agnostic Go package"
- End B: `.abcd/development/brief/glossary/core/transport.md:4` — "The mechanism by which curated context and artefacts are packaged and delivered to an oracle for review or reasoning."

01-platform says 'The core is a transport-agnostic Go package' behind thin front doors, so transport there means the door a user reaches the core through (CLI, MCP, plugin). The glossary's transport entry has one sense: 'The mechanism by which curated context and artefacts are packaged and delivered to an oracle for review or reasoning.' and records no front-door sense. The constraint chapter uses the term in a sense the canonical glossary does not hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 120.
