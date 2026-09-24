---
schema_version: 1
id: "iss-178"
slug: "autonomous-cloud-runs-in-sibling-repos-leaked-harness-attrib"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "manual-capture"
found_at: "autonomous cloud routines / public GitHub artifacts"
related_intents: [itd-107, itd-152]
related_issues: [iss-172]
resolution: "the harness-leak class (live session URL, tool attribution footer) is defined once in the scanner's canonical pattern set and enforced on three wired surfaces: the store-before-commit redactors, abcd lint's privacy rule, and the harness_leak record/docs-lint rule, with the routine policy carried as scanner.OutboundPolicy. A fourth surface exists as a primitive only: ScrubOutbound sanitises an outbound artefact and is tested, but no command or plugin verb calls it, because spc-45 scopes a forge client out; the three wired surfaces judge already-committed or already-stored text, so posting-time protection is the re-read-and-strip policy that OutboundPolicy states (itd-152)"
impact: additive
resolved_by:
  intent: "itd-152"
  spec: "spc-45"
---

Autonomous cloud runs in sibling repos leaked harness attribution footers and a live session URL into public GitHub artifacts: the harness auto-appends a 'Generated with' footer plus a session link when a PR or issue is created, overriding the repos' Assisted-by-only attribution policy (commit messages stayed clean; the leak surface was PR bodies and issue comments, plus GitHub's public edit history retaining the pre-scrub revision). Leak shape only - no session ids reproduced here. Two remedies needed: (a) every autonomous routine prompt must ban session URLs and harness footers in public text AND mandate a post-create re-read-and-strip of every PR/issue/comment the loop creates, because the append happens outside the model's own text; (b) abcd should detect the class - session-URL and harness-footer patterns belong with the shared privacy pattern set (iss-154 family / itd-74 banlist territory) so audit and docs-lint flag them in any committed or posted text.

Widened 2026-08-11 by a second shape, found in this repo rather than a sibling: the footer is not always appended by the harness outside the model's own words — it can be MODEL-AUTHORED, written deliberately into a pull request body by an agent reasoning from a generic default plus a too-narrow reading of the repo's own policy (here, concluding the attribution rule was scoped to the documentation lint's configured roots, when CONTRIBUTING.md and the agent-instruction router together confine tool naming to the disclosure trailer plus two named credit files). Remedy (a) as written addresses the harness-append shape; the model-authored shape additionally needs a read-what-is-prescribed step before any outward text is composed, because a ban an agent has reasoned its way around is not a ban. Remedy (b) is unchanged and covers both shapes, since a post-create re-read cannot tell which of the two produced the text it strips — which is the argument for keeping (b) even once (a) is tightened.