---
schema_version: 1
id: "iss-2608250844259345"
slug: "a-record-s-resolution-frontmatter-is-user-reachable-and-noth"
severity: "major"
category: "process"
source: "agent-finding"
found_during: "v0.6.6 docs-currency release gate 2026-08-25"
found_at: ".abcd/work/issues"
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (M14: both detectors in order, the co-edit rule first and the resolution-vs-body contradiction detector later, as part of the M13 sibling intent planned next cycle)."
---

a record's resolution frontmatter is user-reachable and nothing checks it against the record body it sits on. The resolution field is emitted verbatim by 'abcd capture list --resolved --json', which commands/capture.md dispatches, so it is a shipped surface rather than an internal note — but the rendered site pages carry the record BODY, so a resolution that contradicts its own body is invisible on the site and visible on the CLI. Demonstrated in the v0.6.6 cut: iss-2608250743421381's body was corrected to say identity-check is deliberately NOT added to the ahoy hint, while its resolution field still said the hint advertises it and describes 'adding the identity-check it registers'. The docs-currency gate caught it by running the command rather than by reading the file, after the body had already been fixed. Same class as iss-2608242043243131 (the preflight gate list restated by hand in five places with no test deriving it) in a different field: a claim with more than one representation and no check that the representations agree. Candidate detector: a record-lint rule asserting the resolution field does not contradict the body, or more tractably that a record edited in a commit has its resolution field edited in the same commit whenever the body's claims change.