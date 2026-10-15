---
schema_version: 1
id: "iss-2610120925509505"
slug: "abcd-build-next-max-1-on-a-live-series"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "build-next review round 2 (run-2610110908001704)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/nextseries.go"
remedy: "Refuse any explicitly given count (--max set, including 1, or --until-empty) that differs from the live series' own, by checking whether the flag was given rather than many(); a test runs --max 1 against a live --max 3 series and expects the exit-2 refusal."
resolution: "build next now refuses any named run count (--max 1 included) that differs from a live series' own, at the pick stage, exit 2; a call with no count still continues the series"
impact: fix
resolved_by:
  commit: "f7b9d16de90504af352fd67a68cca18340827ada"
---

abcd build next --max 1 on a live series is silently taken as a series move instead of the documented refusal. NextOptions.many() is UntilEmpty || Max > 1, so the count check at nextseries.go (if n.many() && (n.UntilEmpty != st.UntilEmpty || n.Max != st.Max)) never runs for --max 1; a --max 3 series in progress plus 'abcd build next --max 1' continues the series with exit 0, while commands/build.md says a count other than the one the series began with is refused at the pick stage, exit 2. Found by the ruthless reviewer, round 2 of run-2610110908001704 lane-1.
