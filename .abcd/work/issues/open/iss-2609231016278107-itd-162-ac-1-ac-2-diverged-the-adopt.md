---
schema_version: 1
id: "iss-2609231016278107"
slug: "itd-162-ac-1-ac-2-diverged-the-adopt"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/prepare-this-repo.md"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lane drainFresh of autonomous run A, 2026-09-29): Should the adopt phase offer a secrets and absolute-path commit gate from the binary, as itd-162 ac-2 promised, or does the record amend ac-2 to say the private name guard replaced that gate deliberately?"
remedy: "Waits on ruling D++ (CA6: build the gate, or narrow itd-162 ac-2): if built, give the prepared repository's pre-commit a secrets and absolute-home-path check over the staged diff through abcd's own scanner (a front door onto internal/adapter/scanner, resolved as the managed commit-msg hook resolves its binary), no outside scanner; if narrowed, add an Audit Notes line to itd-162 saying ac-2's gate was replaced by the private name guard. Prove the build with a hook test that refuses a runtime-built token and a home path in a staged file, the narrowing with record-lint."
---

itd-162 ac-1/ac-2 diverged: the adopt phase's commit-gate step used to offer a secrets + absolute-path gate as a pre-commit framework config (commands/prepare-this-repo.md at 328a6755^1 lines 171-174, template under a machine-local ~ path), and the intent promised that asset would resolve from the record or the binary instead. The delivery (PR #555) substitutes a different asset: step 5 now scaffolds abcd's private name guard (internal/core/ahoy/defaults/pre-commit, via ahoy install), and the embedded hook carries no secrets scan and no absolute-path gate, so an adopted repository no longer gets the gate the step existed to offer. spc-54 conflated the two ('the pre-commit config the Phase-3 line reaches for is thus already available embedded'). Either the adopt phase should offer a secrets/absolute-path commit gate from the binary or the record should say that gate was dropped deliberately

## Remedy grounds (2026-09-29)

- The common practice is a pre-commit framework hook calling an external scanner such as gitleaks: https://github.com/gitleaks/gitleaks (consulted 2026-09-29). abcd already carries a secret and PII scanner in its binary, so reusing it avoids a new tool; adopting gitleaks instead would need the person's sign-off.
- Rejected: the pre-commit framework config the old step shipped, which reached for a machine-local template and an outside runtime.
