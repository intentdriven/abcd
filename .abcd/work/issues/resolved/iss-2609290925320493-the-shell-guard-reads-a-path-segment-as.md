---
schema_version: 1
id: "iss-2609290925320493"
slug: "the-shell-guard-reads-a-path-segment-as"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: verify-fix4-guardResid"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
resolution: "globReading compares each target with every run of * written as one * (collapseStars), beside the target as written, so /**, ~/**, $HOME/**, ${HOME}/** and ~/../** block as /*, ~/* and ~/../* do, bare, in sh -c and in bash -c. TestStarRunsReadAsOneStar pins them."
impact: fix
resolved_by:
  commit: "4ef5013bd"
---

The shell guard reads a ** path segment as a name, not as the * every shell without globstar expands it to: rm -rf /**, ~/**, $HOME/**, ${HOME}/** and ~/../** are allowed, although bash 3.2, bash 5.3 and /bin/sh each expand ~/** to the home's entries and ~/../** to the home's siblings with the home among them (echo only), so /** deletes what /* deletes and ~/../** deletes the home. A run of * in a glob matches what one * matches, and with globstar set ** matches more, never less.

## Grounds

- pursued: every rm -r target beginning at the root or a home spelling whose ** segments, read as *, name the root, the home or the home's entries blocks; such a target that still allows, or a target a * reading does not reach that newly blocks, would show it wrong
