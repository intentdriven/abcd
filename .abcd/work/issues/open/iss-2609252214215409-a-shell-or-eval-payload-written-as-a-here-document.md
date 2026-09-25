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
---

A shell or eval payload written as a here-document substitution is not read, though its text is the payload verbatim. sh -c "$(cat <<EOF … EOF)" and eval "$(cat <<'EOF' … EOF)" carry the document's text as the string the shell runs, but the tokenizer leaves a substitution's output unknown, so the payload is uninspectable and the family's loud WARN is the verdict: a blocked command in the document runs (review5-guard finding 3). Where the output is fixed — cat with nothing but the here-document, a quoted delimiter or a body with no expansion in it — the document's text can be read as the payload as well as the unknown output.
