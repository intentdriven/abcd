---
schema_version: 1
id: "iss-2610262035265656"
slug: "the-docs-chapter-ships-the-itd-60"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 572 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/10-docs.md:76"
remedy: "none (filed automatically)"
finding_ref: cfk-98edc4939d947f12
---

The docs chapter ships the itd-60 doc-fidelity gate, saving receipts keyed by commit; universal patterns calls the itd-60 pass a draft that is not yet built, keyed by input fingerprint.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/10-docs.md:76` — "labelled with the commit the checkout stands at, under `.abcd/.work.local/doc-fidelity/<commit>/`."
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:144` — "the doc-fidelity anti-drift pass planned under **draft intent itd-60** (`intents/planned/itd-60-doc-fidelity-anti-drift.md`; not yet built)"

10-docs lists `fidelity` and `fidelity record` as shipped and says the record sub-verb saves the docs review as a receipt under `.abcd/.work.local/doc-fidelity/<commit>/`. The spec close and the release cut both wait on that receipt. 04-universal-patterns describes the same itd-60 pass as planned under a draft intent and 'not yet built', with a different on-disk contract: a content-addressed `doc-fidelity/<input_fingerprint>/` tier under the log tier. Both cannot hold. Someone building from the patterns chapter would treat a gate that already blocks closes and cuts as future work, and would lay out a second, incompatible store for it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 572.
