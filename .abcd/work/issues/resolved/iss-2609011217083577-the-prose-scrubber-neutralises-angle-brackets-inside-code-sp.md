---
schema_version: 1
id: "iss-2609011217083577"
slug: "the-prose-scrubber-neutralises-angle-brackets-inside-code-sp"
severity: "major"
category: "ux"
source: "user-observation"
found_during: "the v0.7.0 cut, changelog ingest"
found_at: "internal/termsafe/prose.go"
origin: researcher-authored
production_mode: hand-written
resolution: "cleanProse leaves CommonMark code spans byte-for-byte and still neutralises every HTML opener and comment close outside one: neutraliseSpanAware draws span boundaries the way a renderer does (a run of N backticks closed by the next run of exactly N, backslash-escaped backticks literal outside a span, a run with no closer literal), so a documented placeholder in a span survives ingest and an unbalanced backtick leaves what follows prose. The cap is followed by a re-neutralisation loop so a cut inside a span cannot expose its content. TestCleanProseLeavesCodeSpansAlone pins the span, double-backtick, unbalanced and escaped cases and TestCleanProseCapCannotExposeASpan pins the cap at every length. The exemption is the HTML rule's alone: the markdown-link rule that landed on main meanwhile (iss-2608311504353427) still fires inside a span, because it defends record-lint's links_resolve gate rather than the render and checkLinks masks fenced blocks only, so a code span is scanned like any other prose; TestCleanProseNeutralisesLinkSyntaxInsideACodeSpan pins that asymmetry. FOLLOW-UP, same change set: the first cut of this fix exempted spans from the WHOLE HTML rule, which let an untrusted verdict field shelter a working abcd-review marker (an HTML comment naming a state and a receipt id) inside backticks and spoof the intent audit's review state, and it assumed a cleaned field stays the string it was cleaned as, which ideate's blockText and two same-line embeddings broke. Corrected before merge: the HTML-COMMENT delimiters (`<!` and `-->`) now fire span or not, the tag exemption is kept for `<[A-Za-z/?]` alone so `--reading-json <path>` still survives, the cleaner never emits an unpaired backtick run (it backslash-escapes one, the CommonMark-faithful choice: an unclosed run already renders as literal backticks), and blockText escapes only an unbalanced leading run. TestCleanProseBreaksCommentDelimitersInsideACodeSpan, TestCleanProseEmitsNoUnpairedBacktickRun, TestCleanProseCappedOutputHasNoUnpairedRun, TestIngestVerdictCannotForgeAReviewMarkerFromACodeSpan, TestIngestedAttestationLineCannotRePairACodeSpan and TestBlockTextKeepsACleanedCodeSpanIntact pin the corrected behaviour. renderEvidence in the same file was the third instance of the re-escaping class and is fixed here too: %q doubled the cleaner's own backslash escape and handed the record a live backtick back, so the quote is delimited with plain quotation marks and TestIngestedEvidenceLineKeepsTheCleanedBytes pins it. The sibling sweep over every CleanProse caller found the same-line-embedding half now closed everywhere by the primitive (no cleaned field can carry an unpaired run), and three renderers that still add their own delimiters to a cleaned value — memory's index, contradictions and match renders, and two lifeboat renderers — recorded as iss-2609020539188868 and out of scope here: a different package, a different caller, and none a defect this change introduced. The whole suite passes."
impact: fix
---

The prose scrubber neutralises angle brackets inside code spans, so a documented
placeholder ships as a shell redirect. `cleanProse`
(`internal/termsafe/prose.go:83`) rewrites every HTML opener by inserting a space
after the `<`, which is correct and load-bearing for prose: it stops an injected
`<script>` or `<!--` reaching a rendered surface as live markup, and the comment
case folds into the same rule. It does not exempt code spans, where CommonMark
never parses HTML in the first place, so the neutralisation buys nothing there
and corrupts the content instead.

Observed while ingesting the v0.7.0 changelog. The composer emitted a code span
holding the invocation with an angle-bracket placeholder:

```
abcd reading ingest --reading-json <path>
```

and `launch ship --changelog-json` wrote a space in after the opener:

```
abcd reading ingest --reading-json < path>
```

The payload on disk is correct; the mutation happens inside the ingest. The
damage is not cosmetic: `<path>` is an inert placeholder, while `< path>` is a
valid shell input redirection from a file named `path`. A reader who copies the
documented invocation out of the release record runs something that means
something else, and fails in a way that does not point back at the changelog.

`-->` has the same shape (rewritten to `-- >` on the next line), so any prose
documenting an HTML comment or an arrow token is exposed the same way.

## Grounds

- deferred: not fixed at the point of discovery, because the finding surfaced
  mid-release-cut and the fix touches a primitive that every store-before-commit
  redactor routes through; the v0.7.0 line was reworded to avoid the placeholder
  instead, which leaves the defect intact and recorded rather than half-fixed
  under release pressure

- pursued: the neutralisation buys nothing where CommonMark parses no HTML, so skipping spans by the spec's own boundary rule is faithful and fails closed on a malformed span; refusing at ingest would have kept the corruption reachable from every other redactor on the primitive's path

## Candidate remedies

Not decided; the choice wants a maintainer.

- Skip the neutralisation inside backtick spans, parsing the span boundaries
  first. Most faithful, and the most code: the scrubber gains a tokenizer, and a
  malformed or unbalanced span has to fail closed rather than open.
- Substitute a lookalike that cannot open a tag, such as the full-width `＜`, or
  an HTML entity where the destination is known to render markdown. Cheap, but it
  silently alters a string a reader may copy, which is the same class of defect
  as the one it replaces.
- Leave the primitive alone and refuse at ingest instead: a changelog payload
  carrying `<` inside a code span is rejected with a message telling the composer
  to reword. Fails closed, adds no rendering knowledge to the scrubber, and keeps
  the correction where the prose is authored.

The third fits the existing shape best. `launch ship --changelog-json` already
re-derives the cut and proves the prose against it, so a payload refusal joins
the completeness bijection rather than forming a new gate, and the composer is
already the component that owns wording.
