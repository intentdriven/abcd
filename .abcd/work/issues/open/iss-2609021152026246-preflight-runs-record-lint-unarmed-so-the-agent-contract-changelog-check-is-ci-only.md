---
schema_version: 1
id: "iss-2609021152026246"
slug: "preflight-runs-record-lint-unarmed-so-the-agent-contract-changelog-check-is-ci-only"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "pr-606-merge-2026-09-02"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
---

make preflight runs record-lint without -agent-diff, so the agent_contract sub-check that asks whether a changed agent prompt bumped its prompt_version and wrote its agents/CHANGELOG.md entry is a no-op locally and arms only in CI, which passes -agent-diff. PR 606 passed a green local preflight three times and was refused in the merge queue for exactly that check: agents/cold-reading-widening.md changed with prompt_version still 0.1.0. Same shape as iss-2608311632382737 (preflight blind to the eval lanes, fixed by PR 607): a gate the push relies on that the local run cannot see. Fix: preflight passes -agent-diff origin/main...HEAD (or the merge-base) to record-lint, and the gate-roster test that pins preflight's contents asserts it.
