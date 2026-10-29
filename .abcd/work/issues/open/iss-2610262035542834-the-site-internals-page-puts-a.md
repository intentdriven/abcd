---
schema_version: 1
id: "iss-2610262035542834"
slug: "the-site-internals-page-puts-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 606 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/10-site.md:109"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Does itd-140 rule 3 (adr-47 decision 6: no repo-agnostic claim until a second sparse instance demonstrates it) still gate the site, now that site setup ships for any managed repository and the 2026-10-14 Pages ruling promises the record for every managed repo? Answers: (1) the gate stands and the Pages intent's second-repo acceptance becomes the demonstration; (2) a new ruling lifts the gate. Recommend (1): it keeps the discipline and needs only prose edits, while the shipped setup verb is already described as what it does rather than as a genericity claim."
finding_ref: cfk-25368db992dc57ff
---

The site internals page puts a second-instance demonstration ahead of any repo-agnostic claim; the site chapter already ships site setup for any repository abcd manages

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/10-site.md:109` — "the gate adr-47 decision 6 puts in front of any repo-agnostic claim, and the reason none is made here"
- End B: `.abcd/development/brief/04-surfaces/22-site.md:48` — "Setting up takes a repository abcd manages from the checkout to a live address"

10-site.md says itd-139, the generic explorer demonstrated on a second instance, is the gate adr-47 decision 6 puts in front of any repo-agnostic claim, and that no such claim is made. 22-site.md describes a shipped `abcd site setup` that takes any repository abcd manages to a live address, and says every repository gets the same pages. The setup verb depends on the genericity that, according to the internals page, has not yet been demonstrated, so it ships ahead of its own gate.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 606.
