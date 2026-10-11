---
schema_version: 1
id: "iss-2609300057467536"
slug: "the-guard-allows-rm-rf-and-rm-rf-which"
severity: "major"
category: "security"
source: "impl-review"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
remedy: "Spell $! in segment.spelled as its number and as nothing (the texts $! and empty), for the arg_values compare alone, leaving the token as it is so kill $! and every other reading are unchanged; grounds: printf of $!/ under bash 3.2, /bin/sh, dash and bash 5.3 prints / in a fresh shell."
resolution: "rm -rf $!/ and its siblings ($@/, $*/, $1/, $_/, $-/, braced and quoted, and $! in a trim pattern) now read as the root: the written spelling holds the empty text beside a parameter that can print nothing, while kill $! and every other reading keep the token."
impact: fix
resolved_by:
  commit: "5e3fec20c"
---

The guard allows rm -rf $!/ and rm -rf "$!"/, which bash 3.2, /bin/sh, dash and bash 5.3 print as / when no background job has run: the tokenizer reads $! as a number that stays the text it is (simpleParamEnd), so the word is the literal $!/ and names nothing, though $! is empty until a job runs in the background. Found in the fix-guardSet sibling sweep; outside unknown.go.

## Grounds

- pursued: every word a parameter that can print nothing leaves as / (or ~, $HOME) blocks as rm-rf-root-or-home, and the 1099-line corpus keeps its verdicts; a shell that prints the empty reading as anything but the text beside it, or an everyday $!/$@ idiom that now blocks, would show it wrong.
