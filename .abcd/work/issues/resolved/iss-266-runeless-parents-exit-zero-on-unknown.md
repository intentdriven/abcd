---
schema_version: 1
id: "iss-266"
slug: "runeless-parents-exit-zero-on-unknown"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "spc-30 build"
found_at: "internal/surface/cli/cli.go"
resolution: "Every cobra parent is now runnable, so its declared Args validator actually runs and refuses an unknown sub-verb at exit 2 instead of printing help at exit 0. Six parents fixed (docs, docs cite, guard, embark, ideate, hook); the two existing inline fixes consolidated onto a shared helpRunE. Refusing needs BOTH halves — a runnable parent with Args nil falls through to cobra's ArbitraryArgs and still exits 0 — so TestEveryParentRefusesAnUnknownSubverb asserts runnability AND a declared validator over the live command tree, rather than a hand-kept list. Most parents are cobra.NoArgs; banlist has a custom validator; capture and intent are deliberately ArbitraryArgs (free-text positional, guarded by their own suspected-typo check)."
impact: fix
---

cobra parents with no RunE are not Runnable, so cobra prints help and exits 0 WITHOUT validating args: an unknown sub-verb reads as success to a script. Confirmed on 'abcd docs nonsense' and 'abcd guard nonsense' (both exit 0); 'abcd memory nonsense' and 'abcd history nonsense' exit 1 because those parents have a RunE. spc-30 fixed the disembark parent (a retired sub-verb spelling must refuse, not silently succeed) by giving it a RunE that prints help, so cobra.NoArgs then refuses a stray token at exit 2. The same one-line fix is owed to every other RunE-less parent — sweep them so no unknown sub-verb anywhere exits 0.