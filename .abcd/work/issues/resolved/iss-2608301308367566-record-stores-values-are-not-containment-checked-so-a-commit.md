---
schema_version: 1
id: "iss-2608301308367566"
slug: "record-stores-values-are-not-containment-checked-so-a-commit"
severity: "major"
category: "security"
source: "user-observation"
found_during: "itd-189-round-2-security"
found_at: "internal/core/lint/config.go"
resolution: "Containment is now checked at config-load time: parseConfig refuses every configured repo-relative path that is absolute, unclean, backslashed, or climbs out with '..', through the canonical fsutil.ValidRelPath rather than a second predicate. record_stores and its fifteen sibling path fields (roots, issues_dir, commands_dir, skills_dir, registry, snapshot, target, receipts_dir, runbook, workflow, glossary_dir, baseline, changelog, intents_root, agents_dir, intents_dir, specs_dir and the index_drift doc/dir pairs) are all swept, so a value that escapes never reaches a filepath.Join."
impact: fix
---

record_stores values are not containment-checked so a committed config walks the record gate outside the repository

Found by the round-2 adversarial security review of build/itd-189.
PRE-EXISTING -- reproduces identically on 932629f9 and on main.

A PR that edits `.abcd/record-lint.json` to `"adr": "../outside/decisions"`
reaches `filepath.Join(repoRoot, ...)` at schema.go:1007 with no `..` or
`IsAbs` rejection, and the gate then reads and echoes `.md` frontmatter from
outside the checkout. Reproduced on both HEAD and base:

```text
../outside/decisions/0001-outside-secret.md:2: [BLOCKER record_schema]
filename claims id 'adr-1' but frontmatter declares 'adr-77'
```

`validateRecordStores` checks only prefix membership and inter-store layout,
not containment.

Mitigating factor, and why this is major rather than critical: the edit is
glaringly visible in the PR diff, and a config change is exactly the thing
review looks at. This is a defence-in-depth gap, not an invisible one.

Remedy: reject any `record_stores` value that is absolute or that escapes the
repo root after `filepath.Clean`.

Sibling: iss-2608301203521317 (the store walk's raw os.ReadFile). Both are the
same underlying shape -- the GATE reads attacker-influenceable paths without
the guard the repo already owns -- and `fsutil.ReadGuarded` closes the read
half of both at one call site.

## Grounds

- pursued: a gate at the config door cannot be forgotten by a field added later, where the six per-site containedRepoPath guards were opt-in and silent when omitted; the anti-vacuity guard (this repository's own record-lint.json and docs-lint.json still load) is what would show the predicate too strict, and a managed repo whose legitimate config spells a path uncleanly ('./docs') being refused is what would show the strictness misjudged.
