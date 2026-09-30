---
schema_version: 1
id: "iss-2609300929557796"
slug: "a-repository-s-abcd-config-json-can-still-take-every-command"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/oracle/config.go"
remedy: "Skip such a repository route with one diagnostic naming the repository file and the offending text, termsafe-sanitised and ASCII-quoted so a lookalike letter shows as an escape, through the Diagnostics path every door already prints, and let the machine's own route to the name apply; keep refusing the same fault in ~/.abcd/config.json, the person's own file. Grounds: ruling CD2 as recorded in 03-configuration.md; shown wrong if a repo-only malformed route still makes ahoy credential exit non-zero."
---

A repository's .abcd/config.json can still take every command that reads the provider configuration down (ahoy credential, ahoy connect, the providers board): a route in oracle.roles or oracle.judgements whose name is not a plain lower-case name (the review probe used a Cyrillic U+0456 in scribe) or whose value is not <provider>/<model> refuses the whole configuration in LoadAPI, which ruling CD2 of 2026-09-29 (other commands keep working) says a repository route must not do. The refusal also echoed the repository-authored name to the terminal unescaped, so a lookalike letter read as the plain name it imitates.
