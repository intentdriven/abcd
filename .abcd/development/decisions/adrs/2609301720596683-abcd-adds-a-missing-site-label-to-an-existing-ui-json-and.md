---
id: adr-2609301720596683
slug: abcd-adds-a-missing-site-label-to-an-existing-ui-json-and
status: accepted
date: 2026-09-30
supersedes: null
superseded_by: null
refines: [adr-47]
related_intents: [itd-2609212103568351, itd-2609212103572513]
related_rfcs: []
related_adrs: [adr-47]
---

# ADR-2609301720596683: abcd adds a missing site label to an existing ui.json and never rewrites one

Typed links: `refines` [adr-47](0047-abcdev-app-rendered-from-this-repository-alone.md)
(decision 2's closed allowlist is untouched: this adds a declared label to the
file, and adds no fallback at render time).

## Context

`site-src/ui.json` is the closed allowlist of interface strings adr-47
decision 2 permits the site generator to add. `LoadUI` decodes it with unknown
keys refused and refuses a declared label the file leaves empty or absent,
naming it (`no text for status.target`), so a blank button never reads as a
rendering fault. `abcd site setup` seeds the file once and never rewrites it,
under the rule its chapter states: a file the repository owns once it exists
is kept.

Two intents of this release cycle each declared new required labels:
itd-2609212103568351 the six `status.*` labels of the Now / Next / Later
block, and itd-2609212103572513 `status.target`. Both carry
`impact: additive`, but a managed repository whose `ui.json` predates them
would see a green `site build` refuse on upgrade until it added the lines by
hand, which is breaking for that surface. The review of the second intent's
lane raised it as a ruling owed before the v0.12.0 cut.

## Decision

The product thinker ruled on 2026-09-30 (ruling TG1), verbatim as relayed:
"(b) ABCD ADDS THE MISSING LABELS: on the next site setup or site build, abcd
adds only the missing required labels (with the default words); the
project's own wording elsewhere in ui.json is never changed. No failure, no
manual step; both intents stay impact: additive."

We therefore make one exception to "a file the repository owns once it exists
is kept": `abcd site setup` and `abcd site build` add to an existing
`ui.json` each label the allowlist declares and the file does not carry, with
the words abcd's own bundled `ui.json` gives it, and name each added label on
stderr, one line per label. Nothing else in the file changes:

- A label the file carries keeps its wording byte for byte. A label declared
  blank is the project's declaration, so it is not rewritten and `LoadUI`
  still refuses it by name.
- The added members go at the end of their block, in the block's own
  indentation and line style; a whole declared block the file lacks is added
  with every label in it. Every byte already in the file stays where it was.
- A file that does not decode against the allowlist, an unknown key included,
  is not touched: the closed allowlist refuses it exactly as before, and
  adding applies to declared keys only. The `forge_names` map is not required,
  so nothing is added to it, and `_purpose` is never rendered, so it is never
  added.
- The file is read as the site's other reads are, so a symlinked or
  non-regular `ui.json` is refused, and it is written atomically through the
  canonical writer, keeping its mode.

## Alternatives Considered

- **(a) A breaking impact.** Keep the refusal and declare both intents
  `breaking`, so the cut derives a major-shaped version and every adopter adds
  the lines by hand. Rejected by the ruling: a manual step for words abcd
  already knows.
- **(b) abcd adds the missing labels.** Chosen: no failure and no manual step,
  both intents stay additive, and the project's wording is never touched.
- **A fallback at render time.** Render a missing label from the bundled
  words without writing the file. Rejected: adr-47 decision 2 keeps the
  allowlist closed and the file the one place the site's added words live; a
  silent fallback would render words the repository's file does not hold.

## Consequences

- An older `ui.json` keeps building across a release that declares a new
  label, and the change it takes is visible: the verb says which labels it
  added, and the file is left modified in the working tree for the person to
  commit (`site setup` names it in its commit step).
- `site build` writes one file outside its output directory, the repository's
  `ui.json`, and only when a declared label is absent.
- A future label is additive for adopters by construction, provided the
  bundled `ui.json` declares its words; a test holds the bundled file to
  declaring every label.
