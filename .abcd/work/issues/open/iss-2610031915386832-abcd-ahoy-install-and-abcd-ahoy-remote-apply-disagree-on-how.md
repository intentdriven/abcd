---
schema_version: 1
id: "iss-2610031915386832"
slug: "abcd-ahoy-install-and-abcd-ahoy-remote-apply-disagree-on-how"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "itd-2610030814013772 step 2 review 2026-10-03"
origin: researcher-authored
production_mode: hand-written
remedy: "Give a refused ahoy install a non-zero exit, after its result is rendered so the reason still reaches stdout, and pick one code for both ahoy verbs: exit 2, the code docs/reference/cli/commands.md gives a refusal on every other verb (build, implement, drain, lab), which means moving remote apply's refused from 1 to 2 as well and keeping aborted where scripts already read it. Resolve iss-2610020705158672 in the same change, and pin both verbs' codes with a CLI test that drives each refusal (symlinked .abcd, stale binary, saved claude_md)."
---

abcd ahoy install and abcd ahoy remote apply disagree on how a refused run exits: remote apply returns exit 1 when its result is refused or aborted (internal/surface/cli/cli.go, the applyCmd RunE, 'A change that did not happen exits non-zero'), while ahoy install renders every core refusal (the committed .abcd symlink hazard, the stale-binary guard, the retired docs.target refusal) with status refused and exits 0, so a script cannot tell an install that wrote nothing from one that landed. iss-2610020705158672 names the stale-binary case alone; this is the asymmetry across every install refusal and against the sibling verb.
