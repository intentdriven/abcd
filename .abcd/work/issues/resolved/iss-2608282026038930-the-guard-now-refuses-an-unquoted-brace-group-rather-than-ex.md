---
schema_version: 1
id: "iss-2608282026038930"
slug: "the-guard-now-refuses-an-unquoted-brace-group-rather-than-ex"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "itd-156 adversarial review follow-up"
found_at: "internal/core/guard/tokenize.go"
resolution: "A bounded brace expander following bash 5.3's algorithm replaces the refusal: everyday groups allow, a group expanding to a hazard blocks under that hazard's entry, and a group past 4096 words or 1 MiB per command line is refused. A differential run against bash 5.3 over about 10,000 random words found no brace mismatch."
impact: additive
resolved_by:
  commit: "98091ec636e58911290a37f2d549df4567a86265"
---

The guard now refuses an unquoted brace group rather than expanding it (itd-156/spc-49 scoped the expander out), so every ordinary shell brace an agent writes is blocked: mkdir -p foo/{a,b}, cp x{,.bak}, rm -rf dir{1..9}. That is the intended fail-closed posture — a word whose argv the guard cannot compute is one it cannot check — but it is a real usability cost on the PreToolUse path, paid on every command that uses a shell convenience nobody meant as a hazard. The scoped follow-up is the bounded expander the intent already names: enumerate the Cartesian product of a group's alternatives (nested groups and {a..z} ranges included) under a hard cap on the number of words produced, check each expansion against the registry, and refuse only when the cap is hit or an expansion matches a blocker. Detector: a corpus of everyday brace commands whose verdicts should be allow; acceptance: mkdir -p foo/{a,b} allows while git push {--force,} origin main still blocks.

## Grounds

- pursued: the guard checks the argv bash builds from a brace group; shown wrong by a word whose expansion differs from bash's (TestBraceExpansionMatchesBash) or an everyday group refused (TestEverydayBraceCommandsAllow)
