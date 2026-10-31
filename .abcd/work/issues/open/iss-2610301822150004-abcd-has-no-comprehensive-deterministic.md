---
schema_version: 1
id: "iss-2610301822150004"
slug: "abcd-has-no-comprehensive-deterministic"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "planning itd-2610301750162657"
origin: researcher-authored
production_mode: hand-written
found_at: "ACKNOWLEDGEMENTS.md"
remedy: "Atomicise acknowledgements: one CSL-JSON entry per source under .abcd/development/acknowledgements/<kind>/<id>.json (id from the shared allocator; custom fields for kind, credited_for, adopted_by record ids, via for a secondary route, consent for a named person); generate ACKNOWLEDGEMENTS.md and references.csl.json from the entries in launch ship as real files, with the regenerate-and-compare check in the cut; state the qualifying rules as tests (an adopted idea, a cited source, a landed tool; the primary source credited, the summary recorded as via; software at its version; licence notices excluded) and enforce them through itd-163's gate."
---

abcd has no comprehensive, deterministic system for acknowledging prior art. ACKNOWLEDGEMENTS.md is one hand-edited file that many agent sessions append to at once, so it conflicts on merge (the one-writer-per-file principle names a multi-writer single file as a merge hotspot), and nothing decides what qualifies for an entry: on 2026-10-30 three intents' cited papers and tools were found never credited. The product thinker proposed one file per acknowledged source in a folder under .abcd/development/, a root file regenerated only when a release is cut, and clear deterministic rules for what qualifies. The state-of-the-art check of 2026-10-30 (.abcd/development/research/notes/2026-10-30-acknowledgements-sota.md) supports all three, with one change: the root file should be a real generated file, not the proposed symlink, since Go module zips omit symlinks, core.symlinks=false checks them out as text, and GitHub and GitLab show the link path instead of the content. Related: itd-163 (citation closure and the acknowledgements mirror gate), itd-126 (one shared bibliography).
