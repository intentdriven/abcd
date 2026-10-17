---
schema_version: 1
id: "iss-2610120601203457"
slug: "guard-refuses-python-heredoc-quote"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "security fix lane B2, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
remedy: "Reproduce with a here-document whose body holds an apostrophe after a python3 - <<'PY' opener (quoted delimiter), find why the body is tokenized as command text, and fix it; prove with a Check test that the line allows and a blocker after the closing delimiter still blocks"
---

The abcd guard hook refused a Bash call holding a Python heredoc with 'unterminated single quote', although bash runs the heredoc body as data: a quote inside a here-document body is document text, which the guard's own help says is not an unterminated quote. Hit by lane B2 on 2026-10-12; the exact line was not kept.
