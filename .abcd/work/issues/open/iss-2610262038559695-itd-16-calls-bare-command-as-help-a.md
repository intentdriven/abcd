---
schema_version: 1
id: "iss-2610262038559695"
slug: "itd-16-calls-bare-command-as-help-a"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 827 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md:49"
remedy: "In itd-16 (scope bullet and the second acceptance criterion) replace 'the universal bare-command-as-help convention' with 'the common convention' and say bare `/abcd:audit` renders state, per the Bare invocation section of 04-surfaces/README.md."
related_intents: [itd-16]
finding_ref: cfk-78635cd4bf919cd6
---

itd-16 calls bare-command-as-help a universal convention; the surfaces register says bare invocation is a convention, not a universal

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md:49` — "status+help only per the universal bare-command-as-help convention"
- End B: `.abcd/development/brief/04-surfaces/README.md:227` — "It is a convention rather than a universal, and the exceptions are where the tree does not yet meet its own discipline."

itd-16 justifies bare /abcd:audit by 'the universal bare-command-as-help convention'. The surfaces register says bare invocation is a convention rather than a universal, that it renders state rather than help, and that its exceptions are listed in one place. The two documents mean different things by the bare-invocation rule.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 827.
