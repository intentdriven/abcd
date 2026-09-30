---
schema_version: 1
id: "iss-2609300057467536"
slug: "the-guard-allows-rm-rf-and-rm-rf-which-bash-3-2-bin-sh-dash"
severity: "major"
category: "security"
source: "impl-review"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
remedy: "Spell $! in segment.spelled as its number and as nothing (the texts $! and empty), for the arg_values compare alone, leaving the token as it is so kill $! and every other reading are unchanged; grounds: printf of $!/ under bash 3.2, /bin/sh, dash and bash 5.3 prints / in a fresh shell."
---

The guard allows rm -rf $!/ and rm -rf "$!"/, which bash 3.2, /bin/sh, dash and bash 5.3 print as / when no background job has run: the tokenizer reads $! as a number that stays the text it is (simpleParamEnd), so the word is the literal $!/ and names nothing, though $! is empty until a job runs in the background. Found in the fix-guardSet sibling sweep; outside unknown.go.
