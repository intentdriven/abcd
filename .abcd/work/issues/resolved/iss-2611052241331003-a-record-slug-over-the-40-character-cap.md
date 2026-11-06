---
schema_version: 1
id: "iss-2611052241331003"
slug: "a-record-slug-over-the-40-character-cap"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "records branch before the release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/recordid/slug.go"
remedy: "Judge the cap at the commit: the pre-commit hook runs one check over every added or renamed record's file name and slug field, naming the record and the allowed length, using the same rule as the test; and read the cap from a per-repository setting in .abcd/config.json, defaulting to 40, which every minting verb, the hook and the tree test share."
resolution: "Every minting verb cuts a slug at the one value records.slug_cap reads, 40 by default and 16 to 200 in .abcd/config.json. A record-lint rule holds committed names to it, and the commit-msg hook, which already builds this checkout's binaries, refuses a hand rename over it before the commit exists, rather than the pre-commit hook the remedy named."
impact: additive
---

A record slug over the 40-character cap is first refused by a test in the preflight's audit packages, so a hand rename that goes over it is caught minutes later, after the commit, and the cap is a Go constant that no repository can change. It happened on 5 November, renaming a decision record: the precheck failed on TestCommittedRecordSlugsWithinTheCap about ten minutes after the commit.
