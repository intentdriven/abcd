---
id: spc-48
slug: scanner-adjacency-galloping-probe-structural-fix
intent: itd-155
---
# scanner-adjacency-galloping-probe-structural-fix

## Summary

Replaces the fixed 512-byte adjacency window in the secret/PII scanner with a
galloping (exponential-doubling) `trueMatchEnd` probe, so a match end is never a
truncation artefact of the window edge. The window grows only while a match keeps
running into its edge and stops the moment the match ends short of the edge or
reaches the real end of line. This is the structural fix designed in the
2026-08-08 DECISIONS entry, and it dissolves the two superseded issues — iss-189
(a trailing boundary satisfied by the artificial window edge → spurious
over-redaction) and iss-190 (a window-capped recovery truncating a token and
breaking the recovery chain) — without a boundary classifier or a
clipped-so-skip special case, and without the round-6 cost regression.

## Scope

In:

- `internal/adapter/scanner/scanner.go` — the `probeAt` closure inside
  `scanAllPatterns` (scanner.go:484–497), where a match end is computed against a
  truncated slice, and `stolenJunctions` (scanner.go:539) where the same fixed
  cap bounds the forward junction search.
- `internal/adapter/scanner/adjacency_test.go` — the iss-189 and iss-190 repro
  shapes as regression tests, plus a cost-class assertion.

Out:

- No change to the pattern set, the probe construction (`adjacencyProbe`,
  `junctionProbe`), or the redaction step.
- `maxAdjacencyBacktrack` (the *backward* bound in `stolenJunctions`,
  scanner.go:449) is retained; the galloping change is on the *forward* extent
  only.
- No boundary classifier and no clipped-so-skip branch — the intent forbids both,
  and the design achieves the fixes without them.

## Approach

**Today's truncation source.** In `probeAt` (scanner.go:484–497) the window is a
single fixed slice:

```
limit := at + maxAdjacencyProbeWindow   // maxAdjacencyProbeWindow = 512, scanner.go:434
if limit > len(line) { limit = len(line) }
window := line[at:limit]
m := probes[j].FindStringIndex(window)
... add(patMatch{j, at, at + m[1]})     // end computed against the truncated slice
```

Because RE2 treats the slice's end as end-of-string, a trailing `\b` in a probe
is satisfied by the artificial edge (iss-189), and an open-ended token longer
than 512 has its `end` pinned at `at + 512` — an artificial end that makes the
next `probeAt(m.end)` and `stolenJunctions` start mid-token, breaking recovery
(iss-190).

**The galloping probe.** Replace the single-shot slice with an
exponential-doubling probe that computes `trueMatchEnd`:

1. Start with a small window `w` (e.g. 64 bytes) at `at`.
2. Run `probes[j].FindStringIndex(line[at:min(at+w, len(line))])`.
3. If the match reaches the window edge — `m[1] == w` **and** `at+w < len(line)`
   — the match may still be growing: double `w` and re-run.
4. Stop when the match ends short of the edge (`m[1] < w`) or the window has
   reached the real end of line (`at+w >= len(line)`). In both stop cases the
   slice end is *not* an artefact: either the match genuinely ended before it, or
   it is the true end of line. Set `end := at + m[1]`.

Because the probe grows only while the match keeps running into its edge, a short
match (ending well before 64 bytes) is found on the first attempt with the same
cost and result as today. A long match doubles a logarithmic number of times, and
each doubled scan is over a prefix the top-level `cp.Re.FindAllStringIndex(line,
-1)` (scanner.go:499) already pays for — so the amortised cost stays in the class
the unbounded top-level match already pays, the round-6 regression the DECISIONS
entry warns against.

**`stolenJunctions`.** Its forward bound `hi := m.end + maxAdjacencyProbeWindow`
(scanner.go:547) is replaced with the same galloping extent, so the recovery
chain is never capped by the fixed 512 and the iss-190 chain stays intact. The
backward bound (`lo`, using `maxAdjacencyBacktrack`) and the resume-at-`cut+1`
step (scanner.go:576, the iss-188 fix) are unchanged.

## How it satisfies each acceptance criterion

- *A match running past the old 512 window is captured whole; its end is never a
  truncation artefact* — the galloping loop doubles until the match ends short of
  the edge or hits end-of-line, so `trueMatchEnd` is the real end. Test: a
  redactable token > 512 bytes; assert the whole match is captured and `end`
  equals the true token end.
- *A short match's result and cost are unchanged* — a match ending before the
  initial `w` is found on the first, smallest scan. Test:
  `TestConcatenatedSecretsBothDetected` and the short-match cases in
  `adjacency_test.go` stay green; a cost assertion mirrors
  `TestAdjacencyProbeStaysLinearOnLongLines`.
- *iss-189 shape produces no spurious over-redaction, without a boundary
  classifier* — the trailing `\b` is now evaluated against the true end of line,
  not the artificial edge, so it is not satisfied by the edge. Test: add the
  iss-189 repro (a trailing boundary met only by the window edge) and assert no
  finding, with no boundary-classifier code introduced.
- *iss-190 shape keeps the recovery chain intact, without a clipped-so-skip
  special case* — the token's true end feeds the next `probeAt(m.end)` and the
  galloping `stolenJunctions`, so the third abutting token recovers. Test: add
  the iss-190 repro (window-capped recovery) and assert every token is detected,
  with no clipped-so-skip branch.
- *A match genuinely still growing doubles only while it runs into the edge,
  within the amortised cost class* — the loop's stop condition (`m[1] < w` or
  end-of-line) bounds growth to genuinely-growing matches. Test: assert the
  number of probe doublings is logarithmic in the match length, mirroring
  `TestAdjacencyProbeWindowIsBounded` / `TestJunctionBacktrackIsBounded` as
  cost guards.

## Decisions

The three prior bug-hunt rounds each BLOCKed on a *local* patch (a boundary
classifier for iss-189, a clipped-so-skip for iss-190) that reached further than
the ambiguity and either over-redacted or regressed cost. The DECISIONS entry
resolved on the structural fix: remove the artificial edge entirely, so neither
local patch is needed. This spec encodes that resolution — the galloping probe is
the single mechanism, and the two superseded repro shapes are its acceptance
corpus.
