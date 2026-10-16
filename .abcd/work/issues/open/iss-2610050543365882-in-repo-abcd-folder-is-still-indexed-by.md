---
schema_version: 1
id: "iss-2610050543365882"
slug: "in-repo-abcd-folder-is-still-indexed-by"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "/abcd board session, Spotlight check after the ~/.abcd.noindex rename"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/.work.local/"
remedy: "By default, take only the machine-written local tier out of the indexer: give .abcd/.work.local (or its scratch, logs and transcripts) a name ending in .noindex, so the committed record (.abcd/development, .abcd/work) stays searchable; a person who wants nothing indexed adds the repo's .abcd folder in System Settings, Spotlight, Privacy (abcd never edits that list, as decided for the home), and one setting turns the default off for a person who wants everything indexed; date a receipt per macOS version as itd-2610030720038073 does."
---

Renaming abcd's home to ~/.abcd.noindex (itd-2610030720038073) keeps the home out of Spotlight, but the .abcd/ folder inside each repository is still indexed: a folder whose name starts with a dot is not skipped, only one whose name ends in .noindex. On 2026-10-05, macOS 27.0, mdfind in this checkout returned at least 45,000 indexed items under .abcd/.work.local and about 3,150 under the committed tiers, and .abcd/.work.local/scratch alone holds about 130,000 files (1.6 GB), so every scratch export, log and review an agent writes there sets off indexing work, the cost the home rename was meant to remove. The intent's scope names the home only, so nothing covers the in-repo tiers.

## Addition 2026-10-16 (product thinker)

The product thinker asked for the same fix and set its scope: abcd excludes the local tier from Spotlight by itself on a Mac, in every repository it manages, not only this one, by giving the folder a name ending in `.noindex` (for example `.abcd/.work.local.noindex/`). The rename needs a matching `.gitignore` entry in each repository, so the renamed folder is never committed by accident, and checkouts that still carry the old folder need it moved. It is not urgent.

The same day showed what the name buys on macOS 27.0.1 (iss-2610161806164689): Spotlight reads no contents of files under a `.noindex` folder but still records each file's name and dates, so a person who wants no entries at all still adds the folder to Search Privacy by hand.
