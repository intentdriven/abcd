---
schema_version: 1
id: "iss-2610222012245116"
slug: "abcd-capture-remedy-iss-n-given-no-fix"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "drain session 2026-10-22"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/capture.md"
remedy: "When capture remedy <iss-N> is given no fix text, the command page has the host set abcd mode product-thinker, read the record (abcd <iss-N> --json), retell what the issue is in plain language with no file paths, function names or jargon, and ask the product thinker for the remedy through the question tool, writing their own words with capture remedy; the binary's refusal for the missing text names that route instead of the bare argument-count error."
---

abcd capture remedy <iss-N> given no fix text only fails with 'requires at least 2 arg(s)': nothing reminds the product thinker what the issue is or asks them for the remedy, so a person who comes to write the fix for an ineligible record (one with no remedy, or 'none (filed automatically)') has to go and read the record themselves, often written in code terms they cannot follow.
