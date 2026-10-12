---
schema_version: 1
id: "iss-2610120138153613"
slug: "the-implement-loop-s-unarmed-conflict"
severity: "nitpick"
category: "documentation"
source: "impl-review"
found_during: "ruthless review of the loop's conflict refusal (iss-2610110908443341, PR #960)"
origin: researcher-authored
production_mode: hand-written
found_at: "the implement loop's conflict refusal (conflictRefusal in internal/core/implement/loop/conflict.go, added by PR #960)"
remedy: "say 'arms it once the forge no longer reports it conflicting', or make the arm wait on UNKNOWN as the wording says"
---

The implement loop's unarmed conflict remedy (internal/core/implement/loop/conflict.go, conflictRefusal's default case) says the next step 'arms it once the forge reports it mergeable', but the arm proceeds on any verdict other than CONFLICTING, UNKNOWN included (land.go's arm path). The wording promises a wait the code does not make.
