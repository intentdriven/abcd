---
schema_version: 1
id: "iss-2610100626320367"
slug: "a-record-s-file-name-has-no-cap-tied-to-the-length-of-its"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "the review of the rewriter proposal, 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "the slug derivation the record-minting verbs share"
remedy: "Cap the slug the minting verbs derive so a record's repository-relative path stays within a fixed budget, cut on a word boundary as the title cap already is, with a test over the record family with the longest prefix; the general rewriter filed alongside can later shorten a title so the cut reads well, but the cap is the guarantee."
---

A record's file name has no cap tied to the length of its full path, so a record checked out on Windows can pass the 260-character path limit, which the product thinker reports records already do. The longest tracked path is 149 characters inside the repository; the checkout's own location comes on top, and a worktree in the machine store adds its prefix, the 40-character root commit and the worktree's name. Git for Windows will not check such a file out unless core.longpaths is set (itd-2610070544145422 is the Windows install draft).
