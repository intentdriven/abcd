---
schema_version: 1
id: "iss-2608250844259345"
slug: "a-record-s-resolution-frontmatter-is-user-reachable-and-noth"
severity: "major"
category: "process"
source: "agent-finding"
found_during: "v0.6.6 docs-currency release gate 2026-08-25"
found_at: ".abcd/work/issues"
remedy: "Waits on the M13/M14 intent's interview (refuse or advise): add a co-edit rule to scripts/check-issue-resolution.sh that flags a commit changing the body of a record under resolved/ or wontfix/ without changing its resolution field; if refuse, it fails the range; if advise, it prints a warning through the shared warn baseline. Prove it with a fixture commit editing a resolved record's body alone (flagged) and one editing both (clean); the contradiction detector follows as the second layer M14 names."
deferred_after: v0.11.1
deferral_reason: "Ruled M14 on 2026-09-23: the co-edit rule first (a body change on a resolved record requires a resolution change in the same commit), the contradiction detector second, both inside the one sibling intent ruling M13 names with iss-2608241612007530 and iss-2608261635558358. That intent is not filed, filing it needs the product thinker's adoption, and its interview owes one question: does the co-edit rule refuse or advise?"
---

a record's resolution frontmatter is user-reachable and nothing checks it against the record body it sits on. The resolution field is emitted verbatim by 'abcd capture list --resolved --json', which commands/capture.md dispatches, so it is a shipped surface rather than an internal note — but the rendered site pages carry the record BODY, so a resolution that contradicts its own body is invisible on the site and visible on the CLI. Demonstrated in the v0.6.6 cut: iss-2608250743421381's body was corrected to say identity-check is deliberately NOT added to the ahoy hint, while its resolution field still said the hint advertises it and describes 'adding the identity-check it registers'. The docs-currency gate caught it by running the command rather than by reading the file, after the body had already been fixed. Same class as iss-2608242043243131 (the preflight gate list restated by hand in five places with no test deriving it) in a different field: a claim with more than one representation and no check that the representations agree. Candidate detector: a record-lint rule asserting the resolution field does not contradict the body, or more tractably that a record edited in a commit has its resolution field edited in the same commit whenever the body's claims change.

## Deferral 2026-09-29

Deferred past v0.11.1: Ruled M14 on 2026-09-23: the co-edit rule first (a body change on a resolved record requires a resolution change in the same commit), the contradiction detector second, both inside the one sibling intent ruling M13 names with iss-2608241612007530 and iss-2608261635558358. That intent is not filed, filing it needs the product thinker's adoption, and its interview owes one question: does the co-edit rule refuse or advise?

## Remedy grounds (2026-09-29)

- The ruling of 2026-09-23 in .abcd/work/DECISIONS.md orders the co-edit rule first, and it is mechanical where a contradiction detector needs judgement.
- The issue-resolution script already walks the range's record moves, so the rule sits beside RS006 rather than in a new gate.
- Rejected: a semantic contradiction check first, which the ruling sequences second.
