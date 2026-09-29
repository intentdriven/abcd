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
---

The shell guard allows recursive deletes of the home directory spelled three ways its arg_values compare does not read: a backslash-newline inside the variable's name (rm -rf $HO<backslash-newline>ME, which bash reads as $HOME and the guard spells ${HO}ME), a variable inside a brace expansion (rm -rf {$HOME,x}, rm -rf $HOME/{.*,}), whose words carry no written spelling, and a parameter expansion of HOME with an operator (rm -rf ${HOME%/}, ${HOME:-x}, ${HOME#}, ${HOME/x/x}, ${X:+$HOME}), whose value can be the home but whose spelling is not one of the words the entry names. rm-rf-root-or-home promises to block a recursive delete of the home wherever it stands, and each of these deletes it. Present at main a018e7ca2 and at 8cd7f88f4.
