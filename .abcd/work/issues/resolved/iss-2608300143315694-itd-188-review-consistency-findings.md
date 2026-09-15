---
schema_version: 1
id: "iss-2608300143315694"
slug: "itd-188-review-consistency-findings"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "itd-188 adversarial review, 2026-08-30"
found_at: "internal/core/lint/scribecontract_test.go, .abcd/development/brief/05-internals/05-prompt-quality.md, .abcd/development/specs/open/spc-66"
resolution: "spc-66's acceptance mapping for criterion 2 is amended to say the criterion is procedural with the retained sessions as evidence and no mechanical test, and its Tests section is brought in line with what ships. 05-prompt-quality.md counts eleven prompts, seven outside the roster. The guard test moves to the external test package and shares preflightgates_test.go's readRepoFile; the second backticked-token regex is gone with the whole-file scan that replaced it; the self-comparing control is replaced by an eight-case bypass table; the canary presence checks the agent_contract rule already runs are dropped; the canary's manifest path is spc-58's home."
impact: internal
---

itd-188 review consistency findings: spc-66 maps acceptance criterion 2 to TestReadingAndScribeSessionsAreDistinctRecords in internal/core/history, which the build neither adds nor declares dropped (deliver it or amend the mapping to say the criterion is procedural); 05-prompt-quality.md still counts ten prompts and six outside the roster after the eleventh landed; the guard test duplicates indexTokenRe and a readRepoFile helper, carries a control that compares a literal to itself, and repeats the canary presence checks the contract rule already runs; the canary's manifest path does not match spc-58's manifest home.
