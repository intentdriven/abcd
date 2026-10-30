---
schema_version: 1
id: "iss-2610262229362789"
slug: "abcd-land-rewrites-its-composed-body"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "review of fix/catch-ci-failures-before-push"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/prbody.go"
remedy: "Name the body file after the branch and its head commit and leave an existing file as it is, saying so in the note; the composition is fixed by the commits, so an existing file at that name is land's own or the person's edit of it."
resolution: "land names the body file after the branch and its head commit and keeps a file already there, judging the kept text and saying so; a new commit names a new file"
impact: fix
---

abcd land rewrites its composed body file under .abcd/.work.local/scratch/land/ on every run, so a person who edited the body before running the printed gh pr create command, then ran land again, loses the edit without a word.
