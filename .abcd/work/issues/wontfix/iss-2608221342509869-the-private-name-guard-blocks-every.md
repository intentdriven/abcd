---
schema_version: 1
id: "iss-2608221342509869"
slug: "the-private-name-guard-blocks-every"
severity: "major"
category: "process"
source: "user-observation"
found_during: "agent-finding"
found_at: "ACKNOWLEDGEMENTS.md"
blocked_by: [iss-2608220150157507]
wontfix_reason: "consequence of the banlist blocker already tracked by iss-2608220150157507; no independent remediation"
---

the private name-guard blocks every commit touching ACKNOWLEDGEMENTS.md because a pre-existing citation in HEAD matches an unanchored pattern; two credits (the layout algorithm, the screenshot tooling) are held as local patches until the anchoring fix lands