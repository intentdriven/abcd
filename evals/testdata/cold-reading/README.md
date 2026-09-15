# The cold-reading fixture corpus

Inert on disk; materialised into a temporary directory per run by
`evals/coldreading_fixture_test.go`, which commits it under an invented identity
in a reserved example domain so the manifest's commit reference resolves.

- `baseline/` — a whole miniature repository state with every plant in its
  canonical home. One sentinel token per warm location class, of the form
  `ABCD-EVAL-SENTINEL-<CLASS>`, so a leak names the class that leaked.
- `holed/` — the negative control. It holds the replacement content for the two
  included files a relocated plant lands in; the relocations themselves are
  declared in the eval's `holes` table, which also names the canonical home each
  plant is removed from. A baseline plus declared holes rather than two full
  trees, so the variants cannot drift apart in everything the hole does not
  touch.
- `refused/` — the refusal corpus, one variant directory per shape the exclusion
  floor cannot REDACT and therefore refuses: a file the include table admits
  whole, carrying an excluded heading in a form the section scan does not report.
  It is separate from the baseline because a corpus carrying one cannot be
  assembled at all, which is the behaviour under test. Its plants carry the
  `ABCD-EVAL-REFUSED-` prefix rather than a sentinel class, because a sentinel
  class is counted against the baseline and this material is never in it.
- `home/` — the fixture `HOME`, carrying a planted session-transcript store. Its
  `ROOT_COMMIT_SHA` directory is renamed to the fixture repository's own
  root-commit sha at materialisation, so the transcript class of brief invariant
  15 is genuinely reachable if the assembler ever walked it.

Nothing here is a live record: the corpus is never scanned by this repository's
own record, docs or site gates, whose roots are the repository's own `.abcd/`
and `docs/`.

## Why the plants sit where they do

Each plant is placed so that some rule of the assembler's contract is
falsifiable by it — the corpus is adversarial per rule, not merely
representative. Four placements are load-bearing and easy to get wrong:

- **An excluded heading needs a home on a record type that travels WHOLE.** On a
  projected record type the projection keeps the heading out whatever the
  exclusion floor says, so deleting that heading's exclusion row leaks nothing
  and the rule cannot be falsified there. Every excluded heading therefore has a
  home in a brief chapter, a discipline or a spec; the copies on the shipped
  intent exercise the projected shape as well.
- **The projection's positive half needs a section that is neither projected nor
  excluded.** Without one, a corpus cannot tell "the projection is positive at
  field granularity" from "the projection is gone and the redaction cleaned up
  after it". That is the `## Residue` section on the shipped, draft and planned
  intents.
- **A rule the assembler enforces by REFUSING needs a shape to refuse.** A leak
  cannot reach one: removing a refusal admits nothing new where there was nothing
  to refuse. That is what `refused/` is for, and it is why the redaction verifier
  went unnamed by the coverage matrix through three review rounds — its call
  could be deleted outright with the lane green.
- **A grain the corpus never exercises is a grain no mutation can reach.** The
  match list, the case fold on the structural deny, a subsection under an
  excluded heading and a block scalar under an excluded key each carry a plant of
  their own for that reason: without one, the mutation that removes the rule
  leaves the fixture manifests byte-identical and every assertion green.

`evals/coldreading_coverage_test.go` holds the matrix that records this, rule by
rule, including the rules this corpus cannot falsify and why.
