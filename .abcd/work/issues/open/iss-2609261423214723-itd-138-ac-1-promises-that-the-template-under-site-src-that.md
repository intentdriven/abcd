---
schema_version: 1
id: "iss-2609261423214723"
slug: "itd-138-ac-1-promises-that-the-template-under-site-src-that"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-138"
origin: researcher-authored
production_mode: hand-written
---

itd-138 ac-1 promises that the template under site-src/ that generates /install.sh also renders README's one-liner, so agreement between the two is structural. Delivered: README.md:106 carries a hand-written sh -c one-liner that nothing renders; TestInstallSurfacesAgree (internal/core/site/installsurface_test.go:158, reading README at line 308) holds it to the template's release prefix, checksum lookup and verifier by assertion, so a drift fails the build but the one-liner is still authored twice. Wanted: either the README block is rendered from site-src/install.sh.tmpl, or the intent's audit records the by-assertion form as the accepted narrowing.
