---
schema_version: 1
id: "iss-2609090642031172"
slug: "scan-deep-coerces-any-non-true-answer-to-off"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "bughunt-triage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
---

The interactive install collects scan_deep through resolveValue with the choice set true|false and then compares the returned string to true, so every other answer becomes false and the deep secret scanner is switched OFF. stdinPrompter.Prompt returns the typed line verbatim and nothing re-checks it against the choices it was handed, so a user who answers yes, y, or Yes to a question about scanning for secrets gets the opposite of what they asked for, with no diagnostic. The three neighbouring slots in the same function abandon the install on exactly this typo rather than persisting it: docs_target and oracle_backend validate through inSet and return a partial install with the comment never persist a typo, and the asymmetry is the defect. The override path applyScanDeepOverride is guarded, so the reachable route is the interactive and collect-missing one, and the gate is narrow: Visibility private, trufflehog on PATH, ScanDeep unset. A silent security downgrade is worse than a refused install, which is the judgement the neighbours already encode. Fix: re-check the answer against the choice set as the neighbours do and return a partial install on anything else, so an unparseable answer never resolves to a weaker scan than the one the operator asked for; consider whether a boolean slot should accept the ordinary spellings of yes and no before it refuses. Detector: resolveValue answering yes for scan_deep must not yield a disabled deep scan, and must take the same partial-install route the docs_target and oracle_backend typo cases take, with the existing false answer still disabling it deliberately. Found while re-scoping iss-33, whose body had recorded this half as fixed; it is not.
