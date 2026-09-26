---
schema_version: 1
id: "iss-2609260115287911"
slug: "the-shell-guard-skips-a-backtick-texts-escaped-substitutions"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
resolution: "A backtick's text is read after bash's backslash pre-pass: a backslash before a dollar sign, a backtick or a backslash (and, directly inside double quotes, a double quote) is removed before the text is followed, so an escaped substitution between backticks, in a word or an unquoted here-document body, is read as the one bash runs; the twelve shapes bash runs now block on the entry the hazard names."
impact: fix
resolved_by:
  commit: "1d5b5d2f"
---

The shell guard follows a backtick substitution's text verbatim, but between backticks bash removes a backslash before a dollar sign, a backtick or a backslash before it parses the command. So in an unquoted here-document body inside backticks, an escaped dollar-paren substitution or an escaped backtick pair is a live substitution bash runs, and the guard skips it as escaped: echo, a backtick, cat <<F, a body line holding \$( <hazard> ), F, a backtick is a silent allow (and its <<-F, x= and double-quoted twins), while bash 3.2 and 5.3 run the hazard (verified with a neutral word). The same pre-pass is missed for an escaped backtick pair nested inside a top-level backtick. Found by review7-guard finding 1; pre-existing at f97a7514.

## Grounds

- pursued: the review's shapes and the nested-backtick twins block while the outside-backtick and quoted-delimiter shapes still allow; a backtick shape bash runs that still allows would show it wrong
