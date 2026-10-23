---
schema_version: 1
id: "iss-2610230016544142"
slug: "directory-level-reservation-footprints"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "lane C, 2026-10-22 crash recovery"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "Make file level the default: a spec's footprint names files, abcd build reserves those files and widens the reservation step by step as a step names more, and abcd reserve warns, naming the directory and the file count it covers, when a footprint names a directory"
---

Directory-level reservation footprints over-reserve and serialise unrelated lanes. On 2026-10-22 four lanes were held up by footprints naming whole directories or packages when their work changed a few files: the worktree store against the merge gate, the sweep-reservations branch holding all of internal/surface/cli against a store build adding two new files there, a release hold against internal/core/landing, and a planning branch holding whole record directories. Each was cleared only by hand-narrowing the footprint to the files the branch changed. Specs' footprint lines and abcd reserve both take a directory as readily as a file, and nothing warns when a directory is named.
