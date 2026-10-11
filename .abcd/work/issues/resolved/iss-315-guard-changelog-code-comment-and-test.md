---
schema_version: 1
id: "iss-315"
slug: "guard-changelog-code-comment-and-test"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "internal/core/guard/payload.go"
resolution: "the false 'loud warn' claim for python/perl is corrected in the CHANGELOG, comment, help and brief; the vacuous test now asserts the real allow posture."
impact: fix
---

guard CHANGELOG code comment and test claim python -c and perl -e already get a loud warn but the shipped behaviour is a silent allow and the pinning test is vacuous
## Evidence
`CHANGELOG.md:126` (released 0.6.0, present tense): "python -c and perl -e carry source this tokenizer cannot read, and guessing a verdict on them would be worse than the loud warn they already get." False: `abcd guard check --command 'python3 -c "...os.system(git push --force)..."'` → `{"verdict":"allow"}`, no warn. Same false claim at `payload.go:186-187` and `interpreters_test.go:105-106`. `TestNonShellInterpretersStayOutOfTheFamily` asserts only `Verdict != Block` — mutation-proven vacuous (passes with python/perl added to isShellFamily).

## Adversarial verdict: CONFIRMED as a doc+test defect (minor)
Two independent refuters confirmed. The silent-allow BEHAVIOUR is defensible per adr-42 (a non-shell interpreter payload is one opaque token; warning on every `python -c` is the warn-storm adr-42 avoids, and the storm is unmeasurable in this Go repo). What is wrong is (a) a released CHANGELOG stating a false posture, (b) the same falsehood at the two sites a maintainer consults, (c) a test that guards nothing. Fix (autonomous, honesty only): reword CHANGELOG:126 + payload.go + interpreters_test.go comments to "recorded but not yet implemented", add the case to the "what an allow does not see" lists (17-guard.md / commands/guard.md / docs/reference/cli/commands.md), and de-vacuum the test with a discriminating fixture asserting VerdictAllow. Implementing an actual warn posture is OUT of scope (needs re-labelling adr-42's discharged corpus evidence). Not prior art: no ledger issue names python/perl.
