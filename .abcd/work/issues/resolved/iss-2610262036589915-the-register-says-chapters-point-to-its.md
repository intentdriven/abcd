---
schema_version: 1
id: "iss-2610262036589915"
slug: "the-register-says-chapters-point-to-its"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 688 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md:248"
remedy: "Soften the register's bare-invocation paragraph: say it is the one complete enumeration of the exceptions and that a chapter may state its own verb's bare behaviour in a line, while the exception-table test keeps the paragraph complete."
finding_ref: cfk-2b065f822d5d3b59
resolution: "The brief's bare-invocation paragraph now says it is the one complete enumeration of the exceptions and that a chapter may state its own verb's bare behaviour in a line, with the exception-table test keeping it complete; it no longer claims the chapters never restate it."
impact: internal
resolved_by:
  commit: "3c6a2ca707829558db5b30b633baf58551b1dac6"
---

The register says chapters point to its bare-invocation list instead of restating it, but the announce chapter restates its bare behaviour

A scope leakage (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/README.md:248` — "This paragraph is the one enumeration of the exceptions; the chapters point here rather than restating it."
- End B: `.abcd/development/brief/04-surfaces/38-announce.md:34` — "Bare `abcd announce` refuses as a usage error naming the kinds."

README says its bare-invocation paragraph is the one enumeration of the exceptions and that the chapters point there rather than restate it. 38-announce states the exception itself: bare `abcd announce` refuses as a usage error naming the kinds. The same behaviour is therefore written in two places, which is the second copy the register says it avoids.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 688.

## Grounds

- pursued: the register no longer contradicts 38-announce's one-line statement of bare announce; TestTheBriefsBareEnumerationClaimsCompletenessNotExclusivity would fail if the exclusivity claim returned or the completeness wording were dropped
