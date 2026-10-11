---
schema_version: 1
id: "iss-2610110946486757"
slug: "kill-search-in-a-string-default"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "Fable review of the shell-string re-read fix (fix/guard-string-reread-gaps)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/payload.go"
remedy: "Feed a substitution the string's own shell runs inside a default the enclosing shell wrote as text (${X:-\\$(pgrep make)}) to the string's commands as the substitution's output, as payloadInput already does for a substitution in the string's own words; one fixture per entry it reaches, watched fail first."
---

The kill-by-search guard entry allows a search inside a double-quoted shell string's default: sh -c "kill ${X:-\$(pgrep make)}" is allowed, although with X unset the string's shell runs kill $(pgrep make). The same search written straight into the string, sh -c "kill \$(pgrep make)", refuses, and so does the direct kill ${X:-$(pgrep make)}. The string's re-read takes only spellings and varLead from the written string, never the feeds of the substitutions it holds (payload.go namedPayloads, payloadInput), so the search's output never reaches kill. main allows it too; it was found by the review of the fix for iss-2610110025455728.
