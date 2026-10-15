---
id: dcn-2609290000000014
slug: ruling-j10-of-the-product-thinker
date: 2026-09-29
---
2026-09-29 — Ruling J10 of the product thinker, applied (lane teachPlane, autonomous run A; Refs: iss-151): itd-103's teaching plane is built rather than dropped from the intent (the ruling, verbatim: "BUILD IT (a lane; the registry is the single source)"). The rules loader bundles a `SHELL` domain generated from the bundled hazard registry the guard reads: one rule per entry and one recall term per command head, plus a fixed five-word list for shell work in general (`bash`, `command line`, `force push`, `shell`, `zsh`); no second copy is written in the bundled `rules.json`, and a test fails the build if the domain and the registry part. Two consequences taken by the lane rather than the ruling: the domain is built from the bundled registry only, never from a repo's `.abcd/guard.json`, because the two features keep independent switches (spc-16, "Config home"); and `SHELL` joins `COMMITTING`, `LOAD` and `PII` as a guardrail domain, so an override that withholds a bundled lesson is named on every load.
