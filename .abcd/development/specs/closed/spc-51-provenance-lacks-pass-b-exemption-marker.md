---
id: spc-51
slug: provenance-lacks-pass-b-exemption-marker
intent: itd-158
---
# provenance-lacks-pass-b-exemption-marker

## Summary

Closes the promised-but-unimplemented claim itd-88's audit found: the lifeboat
press release says Pass B ships as a declared exemption in `_provenance.json`,
never a silent gap, but no exemption field exists in the lifeboat package or the
`Provenance` struct. This spec adds an explicit Pass-B exemption marker to the
`Provenance` record, populates it at the marshal site, and teaches the consumer
to treat a marked record as exempt rather than as an unmarked coverage gap —
while a record with no marker is treated exactly as before.

## Scope

In:

- `internal/core/lifeboat/plan.go` — the `Provenance` struct (lines 65–80) gains
  the exemption field; the marshal assembly (lines 250–259) populates it.
- `internal/core/lifeboat/embark.go` — `readProvenance` (655–673) surfaces the
  marker to the consumer.
- `internal/core/lifeboat/coverage.go` and `embark_render.go` — the coverage
  consumer (`renderCoverageBlanks`, embark_render.go:76–112) recognises an exempt
  Pass B rather than listing it as a blank to answer.
- Tests in `plan_test.go`, `embark_test.go`, `synthesis_principles_test.go`.

Out:

- No change to how Pass B's content is produced or to the probe that marks the
  section blank (`probe.go:677`); this is a marker and its honouring, not a
  change to coverage computation.
- No schema-version bump beyond what the new optional field requires; the field
  is `omitempty` so existing packages round-trip unchanged.

## Approach

**The field.** Add an exemption marker to `Provenance` (plan.go:65–80). Two
shapes are viable; the design uses a small typed field rather than a bare bool so
the exemption can carry its reason:

```go
PassBExemption *PassBExemption `json:"pass_b_exemption,omitempty"`
// PassBExemption{Reason string}
```

The pointer plus `omitempty` makes an unmarked record marshal exactly as today
(the field is absent), which is the load-bearing compatibility guarantee.

**Populate at marshal.** The provenance record is assembled at plan.go:250–259
and marshalled at 260, appended as the `_provenance.json` planned file at 264
(written to disk last, pack.go:322–333). When Pass B is exempt for a package, the
assembly sets `PassBExemption` with its reason; when Pass B is present, the field
is left nil. The manifest-hash exclusion that already keeps `_provenance.json`
out of its own hash (`TestManifestSHA256ExcludesProvenance`, plan_test.go:189) is
unaffected.

**Honour at read.** `readProvenance` (embark.go:655–673) is the single
`json.Unmarshal` into `Provenance`; the field arrives populated or nil with no
code change there. The behavioural change is in the consumer that today treats an
unmarked Pass-B section as a silent gap: `renderCoverageBlanks`
(embark_render.go:76–112) and the coverage aggregation (`coverage.go`,
`Render`/`AlwaysBlank`, lines 97–260). When the provenance carries
`PassBExemption`, the consumer classifies that section as *declared exempt* — it
is reported as an exemption with its reason, not enumerated among the "coverage
blanks — answer these first". When the marker is nil, the section is treated as
`StatusBlank` exactly as today.

## How it satisfies each acceptance criterion

- *An exempt Pass B writes an explicit marker, not a silent gap* — the marshal
  assembly (plan.go:250–259) sets `PassBExemption` when Pass B is exempt. Test:
  build a plan with Pass B exempt, read back `_provenance.json`, assert the field
  is present with its reason (extends `plan_test.go`).
- *The consumer recognises a marked record as exempt* — `renderCoverageBlanks` /
  coverage rendering reclassify a marked Pass-B section. Test: feed a provenance
  fixture carrying the marker and assert the section is reported exempt, not
  listed as a blank (`synthesis_principles_test.go` provenance fixtures,
  lines 31/62).
- *A record with no marker is treated exactly as before* — the `omitempty`
  pointer defaults to nil and the consumer falls through to `StatusBlank`. Test:
  an unmarked provenance fixture yields the identical blank-listing behaviour as
  today.
- *The Provenance struct marshals with the exemption field, closing the itd-88
  claim* — the struct change plus the marshal site. Test: a round-trip
  marshal/unmarshal asserting the field is present and stable (mirroring
  `TestPlanProvenanceRecordsManifestHash`, plan_test.go:213), and a byte-stability
  check that an unmarked record's bytes are unchanged from the pre-change output.

## Decisions

A typed marker carrying a reason, over a bare bool: the press-release promise is
that Pass B is a *declared* exemption, and a reason makes the declaration legible
in the artefact and to the consumer. The field is an `omitempty` pointer so the
change is strictly additive — every existing lifeboat and its tests round-trip
byte-identically, and only a genuinely exempt package emits the marker.
