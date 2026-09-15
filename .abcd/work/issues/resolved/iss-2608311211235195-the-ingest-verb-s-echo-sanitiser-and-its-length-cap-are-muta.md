---
schema_version: 1
id: "iss-2608311211235195"
slug: "the-ingest-verb-s-echo-sanitiser-and-its-length-cap-are-muta"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "TestARefusalNeverEchoesRawPayloadBytes walks the five envelope fields, an item key and the durable refusal record; neutralising the sanitiser or the cap turns it red. The position token, the parked manifest's run id and position, and its assembler version now pass through echo as well."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The ingest verb's echo() sanitiser and its length cap are mutation-vacuous: neutralising termsafe.Sanitize, or the maxEchoedRunes truncation, leaves every test green. Both are trust-boundary guards on a path that echoes model-produced payload text to a terminal and into a durable refusal record. Three interpolations also reach a message unsanitised: ParsePosition quotes the payload's raw position token, and the parked manifest's run id, position and assembler version are interpolated verbatim.

## Grounds

- pursued: a payload cannot rewrite or drown the message that reports its refusal, and the durable record carries no attack rune; a mutation run finding either guard survives would show this wrong
