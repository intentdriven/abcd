---
schema_version: 1
id: "iss-2611060809127037"
slug: "abcd-development-brief-04-surfaces"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/24-decide.md"
remedy: "Say the plain render names the id, title, date and path, and the status, leaving the slug to the path."
resolution: "The Minting an ADR section of the brief's decide page now says the plain render names the id, title, date and path, and the status the record lands with, leaving the slug to the path, in place of the same four values."
impact: internal
resolved_by:
  commit: "289bdc9654184b552ea49c4fcb171a9d90b373af"
---

`.abcd/development/brief/04-surfaces/24-decide.md:97` says the plain render of an ADR mint names "the same four values" as the JSON form and the status. The JSON form quoted at lines 93-94 carries five values: `id`, `slug`, `title`, `date` and `path`. The plain render prints the id, title, date, record path and status, with no slug line of its own.

## Grounds

- pursued: the sentence should match renderDecision in internal/surface/cli/decide.go, which prints the id, title, date, record path and status lines and no slug line; a slug line in that render, or a value it prints that the sentence leaves out, would show it wrong.
