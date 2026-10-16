---
id: dcn-2609020000000011
slug: adr-56-s-third-rule-is-refined-not
date: 2026-09-02
---
2026-09-02 — adr-56's third rule is refined, not contradicted, by the source-and-tests ruling (maintainer): a control that admits an input it cannot examine may pass it only with a per-item unscanned mark in its attestation, so the second rule (an attestation never says more than the examination) carries what the third rule carried; refusal stays the rule for an input the control claims to have examined and could not. Invariant 16 amended to say so. Every ADR is now checked against the framework v4 and the companion v4: adr-58 is superseded by adr-2609021016286571, adr-2609021016272867 is corrected to the derived run, and none of adr-55, adr-57, adr-2609021016275803, adr-2609021016270132, adr-2609021016288378 or adr-2609021016286571 contradicts either document. The two-hundred-thousand-token target is a clarification the documents do not speak to, kept as such.
