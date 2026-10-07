---
schema_version: 1
id: "iss-2610071529246095"
slug: "the-ahoy-install-question-that-sets-what-a-repository"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "abcd ahoy install in a downstream repository on v0.13.2, 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/prompt_help.go"
remedy: "Rewrite the question in plain words: say what a release is to abcd and what the answer changes (what the release preview checks, what the release set-up lays down), and widen the kinds to cover website, native app, mobile app, library or package, command-line tool and agent plugin, plus a generic kind that fits any project and is what 'Decide later' takes; each kind's line says what abcd does for it, and the kinds with no special handling yet say they behave as the generic one; add the new values to artefact.json's accepted set so the release gates can specialise later."
---

The ahoy install question that sets what a repository releases (artefact kind, .abcd/config/artefact.json) gives too little context and too few answers. Its whole explanation is 'What this repository releases; abcd's release commands refuse to guess it.', it offers only plugin, binary and application, and the default is explained as 'as binary today', which tells a person nothing about their own project. A website, a native desktop app, a mobile app or a library has no answer that fits, and nothing says what each choice changes in the release flow. Seen by the product thinker during abcd ahoy install in a downstream repository on v0.13.2, 2026-10-07 (Setup Q11).
