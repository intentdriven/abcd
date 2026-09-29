---
schema_version: 1
id: "iss-2609290625381759"
slug: "the-shell-guard-reads-a-here-document-that-a-command"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
---

The shell guard reads a here-document that a command substitution opens and never reads (x=$(cat <<E), a line, then E) as pending after the substitution closes, so the lines after it are the document's body and allow. bash 5 reads them so, but bash 3.2 and /bin/sh, the /bin/bash and /bin/sh of macOS, drop the document at the close and RUN those lines: echo $(( $(echo <<echo) 1 )) followed by echo B, echo and echo C prints 1, B, an empty line and C there. So echo $(cat <<E) x followed by rm -rf ~ and E allows, and heredoc_test.go pins echo $(( $(cat <<EOF) )) followed by git clean -fd and EOF as allow. Present at main 285455056 and at d53e21cf3.
