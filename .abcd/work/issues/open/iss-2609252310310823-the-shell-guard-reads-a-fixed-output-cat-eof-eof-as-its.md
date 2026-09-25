---
schema_version: 1
id: "iss-2609252310310823"
slug: "the-shell-guard-reads-a-fixed-output-cat-eof-eof-as-its"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
---

The shell guard reads a fixed-output $(cat <<'EOF' ... EOF) as its document's text, but not the same substitution spelled with backticks. A backtick cat of a literal here-document inside sh -c, bash -c or eval double quotes stays an uninspectable payload and warns, and a warn runs; the same backtick substitution standing alone in command position is a silent allow, while bash 3.2 and 5.3 run the document's command in all three shapes (verified with a neutral word). literalHeredocOutput is called only where the substitution opened with a dollar sign (tokenize.go), so the backtick spelling of the review5-guard finding 3 and review6-guard finding 1 shapes is still open.
