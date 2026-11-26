---
id: dcn-2611260616146825
slug: reservation-fork-origin-stays-admitted
date: 2026-11-26
---
2026-11-26 — reservation.fork_origin stays admitted in the repository's .abcd/config.json as the person's ceiling reads it at origin's default branch, and reservation.github_username alone is refused there (technical facilitator, spc-2611251810508418 step 1): dcn-2610161141338923 makes fork_origin a committed, repository-wide setting that names where the repository's own home is, so it says nothing about a person and lies outside the rule that a repository must not say who a person is or where their fork lives; refusing it would refuse every reservation in a repository whose home is a fork, which no ruling of itd-2611251809580662 asked for
