---
schema_version: 1
id: "iss-2610090849280432"
slug: "the-issue-resolution-gate-rs004-does-not"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "2026-10-09 preflight of the records branch"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-issue-resolution.sh"
remedy: "Either read every iss- id on a Refs: or Resolves: line that lists record ids of any family (itd-, adr-, spc-, iss-) as declared, or keep the rule and have RS004 name the near-miss: 'a Refs: line naming iss-N beside an id of another family is not a declaration; put the issue ids on a Refs: line of their own'; add a case for the mixed line, watched fail first."
resolution: "RS004 keeps its two-spelling, issue-ids-only declaration rule and now names the near-miss: a Refs: (or Resolves:) line naming iss-N beside an itd-, adr- or spc- id is refused with 'a Refs: line naming iss-N beside an id of another family is not a declaration; put the issue ids on a Refs: line of their own', pinned by new mixed-line cases in scripts/check-issue-resolution-cases.sh that were watched fail first."
impact: fix
resolved_by:
  commit: "9cc6915f2da530c73b390018031f1e5d62b507c1"
---

The issue-resolution gate (RS004) does not count a Refs: line that names an issue alongside an intent, such as 'Refs: itd-2610090831227812, iss-2610090831317531, itd-146': DECLARE_RE in scripts/check-issue-resolution.sh accepts a line listing iss- ids and nothing else, so the issue on a mixed line reads as an undeclared mention. The refusal then asks for exactly one declaration line, which the commit already appears to carry, and never says the line was passed over because it also names an intent, so the author cannot tell what is wrong. Found on 2026-10-09 when make preflight refused a records commit.

## Grounds

- pursued: a commit or pull-request body carrying 'Refs: itd-146, iss-N' is refused naming the mixed-family near-miss and the fix; a refusal of that line without the sentence, or the sentence on a 'Ref:' near-miss, would show it wrong
