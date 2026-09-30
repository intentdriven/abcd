---
schema_version: 1
id: "iss-2609300651122268"
slug: "rm-rf-root-or-home-over-blocks-a-positional-slice-rm-rf-2-1"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
remedy: "Read a positional or special parameter's slice or part (${@:2}, ${*:2}, ${1:2}) as the parameters it prints, as \"$2\" is read, rather than as a variable's substring that can be the root. Grounds: bash 3.2, /bin/sh and bash 5.3 print the later arguments for \"${@:2}\"; dash has no slice."
---

rm-rf-root-or-home over-blocks a positional slice: `rm -rf "${@:2}"`, `"${@:1}"`, `${@:2}`, `"${*:2}"` and `"${1:2}"` block, though each prints what the parameters hold, as `rm -rf "$2"` (allowed) does. The substring reading, which adds the root, reached positional and special parameters.
