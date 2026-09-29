---
schema_version: 1
id: "iss-2609290925320493"
slug: "the-shell-guard-reads-a-path-segment-as-a-name-not-as-the"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: verify-fix4-guardResid"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
---

The shell guard reads a ** path segment as a name, not as the * every shell without globstar expands it to: rm -rf /**, ~/**, $HOME/**, ${HOME}/** and ~/../** are allowed, although bash 3.2, bash 5.3 and /bin/sh each expand ~/** to the home's entries and ~/../** to the home's siblings with the home among them (echo only), so /** deletes what /* deletes and ~/../** deletes the home. A run of * in a glob matches what one * matches, and with globstar set ** matches more, never less.
