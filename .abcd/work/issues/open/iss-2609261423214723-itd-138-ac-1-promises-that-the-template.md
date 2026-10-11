---
schema_version: 1
id: "iss-2609261423214723"
slug: "itd-138-ac-1-promises-that-the-template"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-138"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (rulings-owed D3, narrowing a shipped promise; run A 2026-09-28): Should README's install one-liner be generated from site-src/install.sh.tmpl (a new writer to a committed file, and the template restructured to hold both forms), or should itd-138 ac-1 be amended to accept the one-liner written by hand and held to the script by TestInstallSurfacesAgree?"
remedy: "Waits on ruling D3: if generated: render README's install block between markers from site-src/install.sh.tmpl and turn TestInstallSurfacesAgree into a byte comparison, so a hand edit fails; if narrowed: record the narrowing in the H10 shape, never rewriting ac-1: this issue resolved by the lane's commit plus an Audit Notes line on itd-138 saying the one-liner is written by hand and held to the template by TestInstallSurfacesAgree."
---

itd-138 ac-1 promises that the template under site-src/ that generates /install.sh also renders README's one-liner, so agreement between the two is structural. Delivered: README.md:106 carries a hand-written sh -c one-liner that nothing renders; TestInstallSurfacesAgree (internal/core/site/installsurface_test.go:158, reading README at line 308) holds it to the template's release prefix, checksum lookup and verifier by assertion, so a drift fails the build but the one-liner is still authored twice. Wanted: either the README block is rendered from site-src/install.sh.tmpl, or the intent's audit records the by-assertion form as the accepted narrowing.

## Remedy grounds (2026-09-29)

Confirmed at this base: README.md line 106 carries the hand-written one-liner and installsurface_test.go asserts agreement by pattern. Rejected: dropping the README one-liner for a link to /install.sh, which removes the pinned, checksum-verified form the intent promises on the front page.
