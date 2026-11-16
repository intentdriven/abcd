---
id: dcn-2611160141179075
slug: the-drive-s-preflight-comparison-reads
date: 2026-11-16
---
2026-11-16 — The drive's preflight comparison reads its own checkout with sparse checkout off (git -c core.sparseCheckout=false) rather than staging with add --sparse, so a preflight checkout that sparse checkout left partial is refused on every git the drive supports; decided by the run for owner review

Grounds: `git add --sparse` first appeared in git 2.34, while the drive runs on git 2.31 and later (it uses `--path-format`). On git 2.31 to 2.33 the flag would make every preflight comparison fail with a misleading read error. Turning sparse checkout off for the comparison's own commands makes `add --update` stage every missing file as removed on every version, and it touches only the drive's scratch index in its own checkout, never the person's checkout or its settings. The patterns file is also held beside `info/attributes`, so a change to it during a run stops the drive.

Reverse it by staging with `add --update --sparse` once the drive's minimum git is 2.34 or later.
