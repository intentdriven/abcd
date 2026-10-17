---
schema_version: 1
id: "iss-2610111014218303"
slug: "series-and-drain-read-open-pr"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "build next lane review, run-2610110908001704, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "The product thinker ruled on 2026-10-12: a pick whose lane waits for a person to merge its pull request counts as done for build next's series, as it does for the drain, so the series moves on to the next pick. Make both verbs read that lane state through one predicate, and add a test where a series lane waiting on its pull request lets the next pick start."
---

Two verbs in the implement loop read one lane state differently: build next's series counts a pick delivered only when its run is Complete(), which a lane whose pull request is left open for a person to merge never reaches, so 'build next --until-empty' in such a repository makes one pick and then only says 'drive run X' until a person merges; the drain reads the same state as DrainLanePullRequest and moves on, because the merge is a person's gate. itd-2609211116005482 decision 4 calls the two flags shared with the drain, and criterion 5's 'its lane stops' does not say which reading is meant. Found by the build-next lane's ruthless review (run-2610110908001704).
