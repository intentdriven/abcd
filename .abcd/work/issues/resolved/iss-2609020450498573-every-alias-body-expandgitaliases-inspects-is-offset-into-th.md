---
schema_version: 1
id: "iss-2609020450498573"
slug: "every-alias-body-expandgitaliases-inspects-is-offset-into-th"
severity: "minor"
category: "observation"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/gitconfig.go"
resolution: "expandGitAliases now offsets each bang-alias body into its own disjoint chain range as it is inspected, from a running chainMax, instead of offsetting every collected body by one shared amount after the loop. A cd in one body therefore no longer reads as preceding an rm in another: 'git -c alias.a=!cd /tmp a && git -c alias.b=!rm -rf . b' allows, matching its sh -c twin, while a cd and an rm in the SAME body still chain and block via rm-rf-after-cd-chain. Proved by TestEachBangAliasBodyGetsItsOwnChainRange (internal/core/guard), watched failing on a scratch archive first."
impact: fix
---

Every `!`-alias body expandGitAliases inspects is offset into the SAME chain range, so two bang bodies on one command line read across each other's cd chain. The bodies are collected in `bang` and offset once after the loop by chainMax+1; each body's own tokenize numbers its chains from 0, so body A's chain 0 and body B's chain 0 land on the same number. An after_cd entry then reads a cd in one body as preceding an rm in another: `git -c alias.a='!cd /tmp' a && git -c alias.b='!rm -rf .' b` blocks via rm-rf-after-cd-chain, while its sh -c twin `sh -c 'cd /tmp' && sh -c 'rm -rf .'` allows, because expandPayloads keeps a RUNNING chainMax and gives each payload its own range. Over-block only, so no hazard escapes; the reading is still wrong. Evidence symbol: expandGitAliases' post-loop `offset := chainMax + 1` in internal/core/guard/gitconfig.go. The fix must give each bang body its own disjoint chain range, as expandPayloads already does for each payload.

## Grounds

- pursued: the reading was wrong in the over-block direction only, but the correct shape is six lines that mirror what expandPayloads already does for a payload, so leaving it open would have cost more to carry than to fix
