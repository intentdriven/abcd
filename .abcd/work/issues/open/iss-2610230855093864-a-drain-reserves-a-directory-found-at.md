---
schema_version: 1
id: "iss-2610230855093864"
slug: "a-drain-reserves-a-directory-found-at"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "review pair of fix/reserve-file-footprints, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Carry the claim result's Directories into the drain's hold and print the same directory warning when a drain lane opens, on stderr in the text form and in the drain's --json, noting that the lane narrows to its changed files at its landing"
---

A drain reserves an issue with the path its found_at names, and when that path is a tracked directory the drain shows no directory warning anywhere: reserveKey builds the drain's hold without the claim result's Directories (internal/core/implement/loop/reservation.go around line 603), and the drain's text and --json renderings never print its start's reservation warnings. The build and reserve verbs warn of a directory footprint; the drain leaves it unsaid, by design so far because a drain narrows each lane to the files it changed once it reaches its landing, but until then the directory holds off every other reservation inside it, unannounced.
