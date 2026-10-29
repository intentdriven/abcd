---
schema_version: 1
id: "iss-2610262032415141"
slug: "launch-uses-documentation-auditor-for"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 303 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:890"
remedy: "In 01-agents.md (design roster row at line 95, and the references at 06-lint.md lines 70-83) rename the unbuilt agent or add a line saying it is not the launch gate row documentation-auditor (the docs-lint engine); the shipped gate id is wire format and stays."
finding_ref: cfk-ea76a5241abd3c58
---

launch uses 'documentation-auditor' for its docs-lint warning gate; the agent catalog uses it for an unbuilt agent that audits docs

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:890` — "a documentation-auditor or hook-compliance warning"
- End B: `.abcd/development/brief/05-internals/01-agents.md:95` — "a documentation audit over a source or lifeboat `docs/` tree, invoked by other verbs rather than by a user"

Launch's acceptance speaks of a documentation-auditor warning, and its gate list defines the documentation audit as the docs-lint engine run over the configured doc roots. The agent catalog uses the name documentation-auditor for a design-target agent, absent from agents/, that would audit a source or lifeboat docs/ tree when other verbs call it. One name covers two different things.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 303.
