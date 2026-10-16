---
schema_version: 1
id: "iss-2610161806164689"
slug: "on-macos-27-0-1-spotlight-indexes-abcd-s"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "abcd-a2 autonomous run, product thinker asked whether Spotlight can skip git history and worktrees"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2610030720038073-abcd-keeps-its-home-folder-out-of.md"
remedy: "Narrow the intent's claim to what holds (contents are skipped, file names are still catalogued), date a receipt for macOS 27.0.1 that counts index entries with mdfind as well as indexer CPU, and tell a person who wants no entries at all to add abcd's home to the Spotlight Privacy list by hand, since abcd never edits that list."
---

On macOS 27.0.1 Spotlight still catalogues every file under abcd's home by name, although the .noindex name keeps it from reading their contents, so the shipped claim of itd-2610030720038073 that Spotlight skips the home holds only in part. On 2026-10-16, after a reboot at about 15:45, mdfind -onlyin ~/.abcd.noindex returned 589,828 entries, 199,371 of them changed that day: lab copies (21,393 in one lab made at 10:04), worktrees, transcripts, scratch and a clone's git objects. A content search for a word that appears in each of the 208 AGENTS.md copies there found none of them, while the same search found the word in a checkout outside the home, so contents are not imported. Each new file still costs the catalogue a little work, which is far less than a content import, but a lab or worktree copy creates tens of thousands at once. The intent's scope condition names macOS 27.0 with a dated receipt and says each later version holds the claim only once its own receipt is dated; 27.0.1 has none, and the receipt that exists measured indexer CPU, not what the index holds.

A repository's .git folder was catalogued the same way (10,689 entries in this checkout) until the person added the folder holding their projects to the Spotlight Privacy list, after which mdfind returned none.

The in-repo .abcd folder is a separate finding, iss-2610050543365882.
