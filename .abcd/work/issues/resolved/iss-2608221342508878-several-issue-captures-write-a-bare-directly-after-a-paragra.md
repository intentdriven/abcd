---
schema_version: 1
id: "iss-2608221342508878"
slug: "several-issue-captures-write-a-bare-directly-after-a-paragra"
severity: "nitpick"
category: "documentation"
source: "user-observation"
found_during: "agent-finding"
found_at: ".abcd/work/issues"
resolution: "record_schema reports a bare --- directly under a paragraph line in a record body, and the six existing captures carrying it are repaired."
impact: fix
resolved_by:
  commit: "48918900"
---

several issue captures write a bare --- directly after a paragraph, which CommonMark renders as a setext heading; surfaced by the record explorer rendering full bodies — the second-detector effect the site intents predicted

## Grounds

- pursued: a --- under a paragraph line is reported while one after a blank line, in a fence, or under a list item or heading is not, and the tree carries none; a capture rendering a phantom heading on a green tree would show it wrong
