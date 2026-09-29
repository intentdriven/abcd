---
schema_version: 1
id: "iss-2609290419119456"
slug: "the-shell-guard-allows-recursive-deletes-of-the-home"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
resolution: "rm-rf-root-or-home reads the home through a backslash-newline inside the name, a brace group's words (a name runs on into a list's or a sequence's letters), a parameter expansion whose operator can leave the value as it is (a subscript read to its matching bracket, with any text after it that holds no alternative at its first operator byte, as bash 3.2 reads it), and an alternative read as its word as written, up to three alternatives deep, split on whitespace where the expansion stands unquoted and with a substitution in it read as its possibly empty output; homeresiduals_test.go pins 249 spellings in TestHomeSpellingsTheWrittenCompareReads."
impact: fix
resolved_by:
  commit: "ad44726df"
---

The shell guard allows recursive deletes of the home directory spelled three ways its arg_values compare does not read: a backslash-newline inside the variable's name (rm -rf $HO<backslash-newline>ME, which bash reads as $HOME and the guard spells ${HO}ME), a variable inside a brace expansion (rm -rf {$HOME,x}, rm -rf $HOME/{.*,}), whose words carry no written spelling, and a parameter expansion of HOME with an operator (rm -rf ${HOME%/}, ${HOME:-x}, ${HOME#}, ${HOME/x/x}, ${X:+$HOME}), whose value can be the home but whose spelling is not one of the words the entry names. rm-rf-root-or-home promises to block a recursive delete of the home wherever it stands, and each of these deletes it. Present at main a018e7ca2 and at 8cd7f88f4.

## Grounds

- pursued: each named spelling blocks bare, in bash -c and (where the outer shell leaves it the home) in sh -c, and 12,595 pre-existing inputs change only on the two pins this fix flips; a home-deleting spelling of those three shapes that allows, or an old input that loosens, would show it wrong
