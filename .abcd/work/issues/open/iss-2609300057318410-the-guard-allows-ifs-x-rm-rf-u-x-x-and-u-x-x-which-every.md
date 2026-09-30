---
schema_version: 1
id: "iss-2609300057318410"
slug: "the-guard-allows-ifs-x-rm-rf-u-x-x-and-u-x-x-which-every"
severity: "major"
category: "security"
source: "impl-review"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
remedy: "On a line that names IFS in any word at any payload layer, an unquoted expansion whose text the guard reads (a default or alternative word, a trim or replacement text, or HOME or PWD) reads as a capped spelling, which refuses rm alone; a variable of unknown value is not capped, so while IFS= read -r d; do rm -rf $d; done stays allowed. Capping is smaller than modelling which assignment reaches which expansion, as splitAfterIFS already rules; grounds: printf under bash 3.2, /bin/sh, dash and bash 5.3."
---

The guard allows IFS=x; rm -rf ${U:-x/x} (and ${U:+x/x}), which every shell splits on the assigned IFS into "" and /, so rm deletes the root; spellWord splits an unquoted default or alternative word on whitespace only (review-guardSet MAJOR-2). IFS=Uv; rm -rf $HOME/x splits the home into / the same way.
