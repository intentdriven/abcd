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
resolution: "Three variants beside enumFlag (enumListFlag for a comma-list string, enumSliceFlag for a string slice, enumSegmentFlag for a repeatable flag whose set applies to one segment of each value) record the closed set on the flag; the facts gain FlagFact.Each and move to format version 2, and abcd explain prints 'each one of' or 'each value's tier one of'. ahoy install --readme-parts and --install-tool and the tier --route are converted, each reading the slice its reader checks; a test holds every flag carrying Each to a reader that refuses a value outside its set."
impact: additive
resolved_by:
  commit: "014783d7f"
---

List-valued flags whose members come from a closed set, such as the comma-list ahoy install --readme-parts and the repeatable --route flags, whose tier segment comes from a closed set, carry no allowed values in the command facts, so abcd explain prints them as a bare string where a single-valued closed-set flag prints its choices
