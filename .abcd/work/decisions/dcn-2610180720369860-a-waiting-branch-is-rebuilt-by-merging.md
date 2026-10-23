---
id: dcn-2610180720369860
slug: a-waiting-branch-is-rebuilt-by-merging
date: 2026-10-18
---
2026-10-18 — A waiting branch is rebuilt by merging, not restacking (facilitator-on-delegation, itd-2610170634394993 planning prep): once its dependency lands, the dependent branch takes the default branch merged in and is never rewritten, because the merge queue lands by merge so the dependency's commits keep their ids, and a restack would break the reachability of resolved_by stamps (RS002, RS003); a restack is used only when the dependency was squash- or rebase-merged by hand, which the review ruleset allows; each rebuild needs a fresh preflight receipt and so takes a load slot (grounds: .abcd/work/rulesets/main-protection.json; main-review.json; AGENTS.md RS002 and RS003; git-spice)
