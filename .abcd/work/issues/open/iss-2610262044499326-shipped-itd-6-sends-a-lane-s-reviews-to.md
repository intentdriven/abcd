---
schema_version: 1
id: "iss-2610262044499326"
slug: "shipped-itd-6-sends-a-lane-s-reviews-to"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1232 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:24"
remedy: "none (filed automatically)"
related_intents: [itd-6, itd-2609170822093401]
finding_ref: cfk-678bc9109e93d181
---

Shipped itd-6 sends a lane's reviews to RepoPrompt through `oracle.review`; itd-2609170822093401 says nothing reads the oracle value and builds a separate routing table for each agent

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:24` — "`oracle.review = rp` opts a lane's reviews in"
- End B: `.abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:26` — "backend is one repo-wide value that nothing reads, and every delegated step"

itd-6 says the built route reads `oracle.review = rp` to opt a lane's reviews into RepoPrompt. itd-2609170822093401 starts from the premise that nothing reads the oracle backend value. It then adds its own repo, machine and bundled routing rows, plus `--route`, for the same reviewer agents. Two documents decide which backend serves a lane's reviews, through two configuration surfaces. Neither names the other or says which one wins.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1232.
