---
schema_version: 1
id: "iss-2610040758485431"
slug: "four-gaps-in-the-documentation-check-s-banned-names-support"
severity: "minor"
category: "ux"
source: "managed-repo"
found_during: "downstream brief-authoring lab report, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/config.go"
remedy: "Add an include file per removable family, document the (?-i:...) form on the lint and banlist pages, require a reason on the allow marker, and decide whether a warning ever escalates; route the --adjudicate proposal through the decision adapter of itd-2609221009495079 rather than a new path; test per part: an included family loads and removes as one, and an allow marker without a reason is reported."
---

Four gaps in the documentation check's banned-names support, met by a downstream project: (1) a name family the repository may later remove has no include file, so its entries sit inline in .abcd/docs-lint.json and removing the family means deleting each by hand; (2) a case-sensitive entry works through the regex flag (?-i:...), but no command page, user doc or brief chapter mentions it; (3) a warning-severity entry never fails any gate, so a warning has no route to being acted on; (4) the docs-lint allow marker records no reason, so an allowed line carries no account of why. The lab also proposed a docs lint --adjudicate mode in which a judge model rules on each match in context: in its own measurement a frontier model judging in session agreed on 97.8% of 280 held-out lines and missed none, and locally run judges reached about 90%.
