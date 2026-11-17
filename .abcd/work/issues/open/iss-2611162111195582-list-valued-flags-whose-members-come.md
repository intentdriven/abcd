---
schema_version: 1
id: "iss-2611162111195582"
slug: "list-valued-flags-whose-members-come"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "after-merge intent audit re-run of itd-2610301346085246"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/facts_test.go"
remedy: "add list variants beside enumFlag, one for a comma-list flag such as --readme-parts and one for a repeatable flag such as --route whose closed set applies to one segment of each value, so each records the set its values are checked against and the facts and help read that one slice"
---

List-valued flags whose members come from a closed set, such as the comma-list ahoy install --readme-parts and the repeatable --route flags, whose tier segment comes from a closed set, carry no allowed values in the command facts, so abcd explain prints them as a bare string where a single-valued closed-set flag prints its choices
