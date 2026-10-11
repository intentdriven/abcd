---
schema_version: 1
id: "iss-2609090951295881"
slug: "ideate-enum-refusals-echo-the-raw"
severity: "nitpick"
category: "security"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ideate/record.go"
resolution: "Fixed: the six closed-set refusals in the ideate recorder (verdict, leg kind, claim status, grill relation, kill outcome, and a grill hit's record id) describe the refused value by its length and never quote it, through termsafe.DescribeRefused; the refusal still names the field and the admitted set. The same shape in lifeboat review and intent audit was captured and fixed as iss-2609290033521472."
impact: fix
resolved_by:
  commit: "a182bb7ca"
---

The ideate verdict recorder states its redaction discipline in the stage-two refusal type: a refusal that quoted the span it refused would publish the leak into the error message, and from there into a terminal and a log, so that error names the finding kinds and nothing else. The enum refusals in the same file do the opposite. Five sites, the verdict, the leg kind, the claim status, the grill relation and the kill outcome, interpolate the offending value into the message after passing it only through the terminal-escape cleaner, which strips control sequences and does not redact. Every one of those fields arrives in the host payload alongside the free-text fields the recorder does redact field by field, so a payload whose status field carries a token or a home path is refused with nothing written and that value printed verbatim to the error surface, where the session transcript and any capture outside abcd take it raw. Verified by reading the five call sites. It matters because the file argues at length that a fail-closed promise is only honest if the refusal itself carries nothing, and these five refusals are the exception nobody wrote down. Fix direction: describe the offending value rather than quoting it, since its length and whether it is empty is enough to fix a typo, or route it through the same field redactor the prose fields already use before it reaches the message. Detector: an ideate payload whose enum field carries a token-shaped value must be refused without that value appearing anywhere in the error text.

## Grounds

- pursued: no ideate refusal carries a refused closed-set value; a payload whose enum field holds a sentinel that appears in the error would show it wrong
