---
schema_version: 1
id: "iss-111"
slug: "the-open-questions-marker-pattern-admits"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "itd-95 P1 review"
found_at: "internal/core/lifeboat/sources_conventions.go"
resolution: "Marker pattern now splits by trailing boundary: TODO/FIXME require ':' or '(', XXX/HACK/BUG keep the bare word (option 2 of the two recorded). On this repo the durable-record documentation FP corpus (scope: .abcd/development/) dropped from 14 to 3 (the 3 survivors are prose that literally quotes the colon form, irreducible for this mechanism); 100% of the bare-word 'mentions a marker' FP class killed. All colon-form fixture markers preserved; only the bare 'FIXME check this' and '# TODO' pinning spellings reclassified to reject, correctly, as indistinguishable from prose. spc-12 pattern description amended in the same change."
impact: fix
---

The open-questions marker pattern admits a bare uppercase word followed by whitespace, so prose that merely mentions a marker matches. Measured against this repository the adapter reports 42 markers across 11 files at medium confidence, and every one is documentation about markers rather than a work marker. The precision cost lands on repos that document their own conventions. Options: require the trailing colon or parenthesis, or drop the bare-word alternative for the ambiguous markers only. spc-12 fixes the current pattern, so this is a design revisit.