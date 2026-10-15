---
id: dcn-2610100000000004
slug: a-target-s-refusal-is-an-error-in-bare
date: 2026-10-10
---
2026-10-10 — A target's refusal is an error in bare `abcd lint`, whatever its rule's findings weigh (iss-2610100649479892). The 2026-07-13 warn mapping stays for findings: re-raising a docs blocker or a site gate failure would double-gate one check. A refusal is not a finding: a docs-lint config that will not load or that the engine refuses (a `roots` entry that does not exist), a site that will not render, a positioning registry or identity block that cannot be read each checked nothing, and the target alone exits 2 on it, so its rule raises one `error` finding naming the target and quoting the refusal, and the aggregate exits 2. `docs-currency` also runs wherever `.abcd/docs-lint.json` exists, not only where `docs/` does, since roots need not include `docs/`; skipping it there hid the refusal entirely. A repo with `docs/` and no config stays a warn: that is a repo that never adopted the docs target, not one whose target refused.
