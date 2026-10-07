---
schema_version: 1
id: "iss-2610050259118177"
slug: "launch-ship-reads-the-release-s-records-at-head-shippedsince"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "v0.13.0 release cut, 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch"
remedy: "Make the derivation refuse a working tree whose record folders differ from HEAD (the same dirty-tree check the ingest runs, naming the paths), or read the records from the working tree and say so; a test runs ship with an uncommitted close and asserts the refusal."
resolution: "the derivation refuses (uncommitted-records, exit 1) a working tree whose terminal record folders differ from HEAD, naming every path; it reads the tree through launch.DirtyTreeFiles, the dirty-tree gate's own reader, so launch ship's emit and ingest and abcd changelog all carry it"
impact: fix
---

launch ship reads the release's records at HEAD (ShippedSince lists them with git ls-tree HEAD), so a spec close left uncommitted in the working tree is neither refused nor included: during the v0.13.0 cut, `launch ship --json` with the home-rename spec's close uncommitted returned rc=0 and a plausible cut of 54 records that silently left out the shipped intent itd-2610030720038073. The dirty-tree refusal fires only on the ingest (--changelog-json), not on the derivation the person reads first.
