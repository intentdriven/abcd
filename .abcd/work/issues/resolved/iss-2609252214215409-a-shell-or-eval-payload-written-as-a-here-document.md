---
schema_version: 1
id: "iss-2609252214215409"
slug: "a-shell-or-eval-payload-written-as-a-here-document"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/payload.go"
resolution: "A word that is wholly \"$(cat <<DELIM … DELIM)\" with a literal body is also read as the document's text wherever it is a payload (sh/bash -c, eval, su -c, env -S); the unknown reading and its warn stay, so the change only adds blocks."
impact: fix
resolved_by:
  commit: "14bb6354"
---

A shell or eval payload written as a here-document substitution is not read, though its text is the payload verbatim. sh -c "$(cat <<EOF … EOF)" and eval "$(cat <<'EOF' … EOF)" carry the document's text as the string the shell runs, but the tokenizer leaves a substitution's output unknown, so the payload is uninspectable and the family's loud WARN is the verdict: a blocked command in the document runs (review5-guard finding 3). Where the output is fixed — cat with nothing but the here-document, a quoted delimiter or a body with no expansion in it — the document's text can be read as the payload as well as the unknown output.

## Grounds

- pursued: a blocked command in the document of sh -c or eval blocks under its own entry, while a clean document keeps its warn and the same document where no shell runs it allows; it would be shown wrong by a literal-document payload whose hazard allows, or by an expanding body read as literal
