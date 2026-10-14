---
schema_version: 1
id: "iss-2610141647307807"
slug: "a-grill-whose-search-silently-failed"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "lab lab-261014142726-9c4d759 (a hand-run of /abcd:ideate on one external write-up, 2026-10-14)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ideate/ideate.go"
remedy: "Deliver itd-2610141646454347: the grill leg records each search (terms, paths, hit counts), and an empty grill without that trace is refused."
---

A grill whose search silently failed writes the same ideate record as an honest empty grill: a search loop that zsh did not word-split returned zero hits for every term (90 files mention one of them), and the verdict schema carries hits only, with nothing saying what was searched or where (lab finding F-4).
