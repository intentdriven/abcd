---
schema_version: 1
id: "iss-2610101903385281"
slug: "two-siblings-of-the-core-hookspath-advice-that"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "abcd-60 drain run 2026-10-10, lane for iss-2610080546210831"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/CONTRIBUTING.md"
remedy: "Give CONTRIBUTING.md and the commit-msg hook's refusal the same three-way advice as commands/prepare-this-repo.md step 5 (unarmed: set core.hooksPath; foreign: have the global dispatcher call .githooks/; armed: nothing), and extend prepare_hookspath_test.go's check to both files."
---

Two siblings of the core.hooksPath advice that prepare-this-repo now makes conditional still give it unconditionally. .github/CONTRIBUTING.md (around line 64) tells every clone to run git config core.hooksPath .githooks, and the commit-msg hook's 'no abcd source' refusal (.githooks/commit-msg around line 162) prints the same command. On a machine whose global hooks dispatcher is in force (ahoy's banlist.hooks_path reports foreign), either one overrides that dispatcher instead of having it call .githooks/. The 2026-10-10 drain lane's reviewer found them; that lane's remedy did not cover them.
