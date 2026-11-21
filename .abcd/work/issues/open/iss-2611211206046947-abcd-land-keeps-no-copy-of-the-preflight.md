---
schema_version: 1
id: "iss-2611211206046947"
slug: "abcd-land-keeps-no-copy-of-the-preflight"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "step 3 of spc-2611012047503720, run-2611211154185741 lane-1"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/land.go"
remedy: "Have abcd land save the preflight's whole output for each head it lands, as a log under the machine's ~/.abcd.noindex/ keyed on the root-commit SHA (outside the worktree, which the build loop removes), named by the head, so cmd/reuse-measure can read every landing this account made; gathering the other accounts' reports stays the open question dcn-2611202032152249 names."
---

abcd land keeps no copy of the preflight report it runs, so a landing's reuse counts and check time can be read back only where its caller saved the output. Read on 2026-11-21 with cmd/reuse-measure over the last ten merges on main (#1721 to #1730): this account's local tier holds a passing preflight log for three of them (#1724, #1728, #1729), each a file a session wrote by saving its own preflight's output, and none for the other seven (#1721, #1722, #1723, #1725, #1726, #1727, #1730), whose preflight either ran on another account or ran here with its output kept nowhere. land.go's makePreflight writes the run's output to the caller's writer and nowhere else, and the push receipt it mints holds the reuse counts and no time, and is lost with the worktree. Criterion 6 of itd-2611011325421304 needs ten consecutive landings' reports, so until each landing keeps its own, the measure cannot be completed from any one checkout.
