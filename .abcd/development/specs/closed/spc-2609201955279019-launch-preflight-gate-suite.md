---
id: spc-2609201955279019
slug: launch-preflight-gate-suite
intent: itd-65
origin: researcher-authored
production_mode: hand-written
---
# launch-preflight-gate-suite

## Summary

This is a REMAINDER spec. A review on 2026-09-20 against the v0.9.0 tree found
the intent about half delivered; the product thinker ruled that the delivered
half is recorded on the intent (its `## Delivery Status` section) and this spec
scopes only what is still open. The fidelity audit reads the intent's criteria
against this spec plus the evidence the intent already cites.

## Scope

Six pieces, each answering one unmet criterion, all on the wired hard-fail path
(`PrecheckPayload` in `internal/core/launch/render.go`) so a preview and a cut
refuse on the same findings:

1. **Marker-block sanity** over every shipped Markdown file: a `<!-- BEGIN ABCD
   -->` without its `<!-- END ABCD -->`, or a nested pair, is a hard-fail
   finding naming the file and line. Today `dryrun.go` reports this gate as
   `not_implemented`.
2. **Change-narration detector** over shipped doc bodies: a deterministic scan
   for the constructs that narrate a change to abcd itself ("changed from X to
   Y", "no longer", "migrated from", "previously … now …" in one sentence),
   hard-fail, naming the sentence. Distinct from the present-tense warning
   `docs lint` already carries, which is advisory. No reroute into the
   changelog: the changelog is derived (adr-37), so the auto-append half of the
   original criterion is moot and is recorded as such on the intent.
3. **Dirty-tree refusal** on the render path, or a recorded decision that
   CI-side tagging of the merged sha makes it moot. The spec's default is the
   refusal: `AllowDirty` is carried and never read (`ship.go`), so either the
   flag gets its check or the flag goes; a flag that does nothing is the shape
   loud-staging forbids.
4. **Warn-fail rows** for the doc-auditor and hook-compliance gates, or their
   removal from the report: a row reported as `not_implemented` on every run is
   a stage that no-ops without saying what it would have checked.
5. **The preflight report** written under `.abcd/.work.local/logs/` on every
   preview and cut, the file the brief says no shipped path writes yet.
6. **A multi-gate test** planting findings in three gates at once and asserting
   the report carries all three: the run-all-collect-all criterion is claimed by
   `wouldRefuseOn` and pinned by nothing.

## Out of scope

The identity hard-fails, the fail-closed scanner, and the local-versus-org
identity distinction are delivered and cited on the intent. Anything that
publishes: publishing is CI's (adr-37).

## Approach

Each piece is a gate function beside its siblings in `internal/core/launch/`,
registered where `dryrun.go` enumerates gates so the preview and the cut see
it; each has a test watched red before the gate exists. The report writer is
one function both callers share (one canonical primitive).

