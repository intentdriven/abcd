---
schema_version: 1
id: "iss-2610100626320367"
slug: "a-record-s-file-name-has-no-cap-tied-to"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "the review of the rewriter proposal, 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "the slug derivation the record-minting verbs share"
remedy: "Cut every minted slug at 40 characters (capture, intent, spec and decide alike), and rename every existing record file whose slug is longer, 2,289 of 2,760 at the ruling, updating every link to them in one change landed when no open branch touches the records; ids do not change. A test holds each record family's longest path within its budget. Ruled by the product thinker on 2026-10-10 ('Tighter cut for all', then 'Rename all 2,289' once told the scale). The general rewriter (itd-2610100627454580) can later shorten a title so the cut reads well; the cap stays the guarantee."
resolution: "Every minted slug is now cut at 40 characters (capture, intent, spec and decide alike), and every record minted before the cap is renamed to it, 2,332 files, with every link to them rewritten; ids are unchanged, and a test holds every committed record's slug within the cap."
impact: fix
---

A record's file name has no cap tied to the length of its full path, so a record checked out on Windows can pass the 260-character path limit, which the product thinker reports records already do. The longest tracked path is 149 characters inside the repository; the checkout's own location comes on top, and a worktree in the machine store adds its prefix, the 40-character root commit and the worktree's name. Git for Windows will not check such a file out unless core.longpaths is set (itd-2610070544145422 is the Windows install draft).
