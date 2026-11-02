---
schema_version: 1
id: "iss-2611020148533389"
slug: "the-implement-loop-s-land-stage-puts-a"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "landing build run-2611012152474713 lane 1, 2026-11-02"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "When the land stage composes a pull request's title or body, add a Refs: line for every issue id the title or body names that no Resolves: line already covers, and test it with a step title that names an issue the lane does not resolve."
---

The implement loop's land stage puts a spec step's title into the pull request's title, and a step title that names an issue (for example '… (resolves iss-2610142036270003)') then names it in public text with no matching declaration. On 2026-11-02 #1344 was opened titled 'build(itd-2610142145236613): The project route carries its content policy in each page and emits no Cloudflare files (resolves iss-2610142036270003), step 1 of spc-2610142146250176'. Its body carried no Resolves: or Refs: line for that issue, because the lane's receipt named no resolves. RS004 refuses a pull request whose title or body names an issue without one of those lines, so the loop's own pull request would fail its record gate. A Refs: line was added to the body by hand.
