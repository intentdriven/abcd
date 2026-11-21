---
schema_version: 1
id: "iss-2611211631294889"
slug: "on-the-interactive-check-and-hook-path"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "review-of-guard-coverage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Read the code a zsh glob qualifier runs the way the guard reads an eval string"
---

On the interactive check and hook path the guard does not read zsh glob qualifiers that run code (e qualifiers and +function qualifiers), so a command it would refuse is allowed inside one.
