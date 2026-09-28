// Release-gate semantic detector — the iss-35 brief↔surface cross-check.
//
// This is a HOST-RUN agent-harness workflow, not a CI step and not a standalone
// executable: it spawns LLM checker agents, so it runs in the maintainer's agent
// harness at release time. The deterministic gates run in CI (see
// ../../../.github/workflows/release.yml); this is the semantic half. It reports
// DISCREPANCIES between the design brief's surface prose and the shipped binary's
// actual behaviour; the maintainer records the verdict as a signed, sha-keyed
// VSA-shaped receipt (see this directory's README.md and the design of record,
// ../plans/2026-07-11-iss35-semantic-release-gate.md).
//
// Invoke with args = the pinned input manifest
// (.abcd/development/release-gate/manifest.json): { briefDocs, surfaces, prompt,
// ... }. The manifest pins the run's inputs (iss-122) — the doc list, the
// directions, the checker count, and the prompt (context + both direction
// templates) are fixed there rather than composed ad hoc here, so two honest
// runs of the same tier examine the same scope. It cannot pin the findings: the
// checkers are LLM agents, and their count and classes vary between runs on
// identical inputs (iss-2608231409595789). The receipt echoes the manifest's
// sha256 as manifestHash and the depth it ran at as tier; receipt_gate refuses a
// receipt whose manifestHash mismatches or whose tier is too shallow for the
// release's impact class.
export const meta = {
  name: 'iss35-brief-surface-crosscheck',
  description: 'Bidirectional brief↔surface reconciliation detector (iss-35 semantic gate)',
  phases: [
    { title: 'CheckBrief', detail: 'per brief doc: verify every surface claim against the binary/tree' },
    { title: 'CheckSurface', detail: 'per real surface: find its brief home' },
    { title: 'Merge', detail: 'dedup + classify discrepancies' },
  ],
}
const input = typeof args === 'string' ? JSON.parse(args) : args
const FINDINGS = {
  type: 'object', additionalProperties: false,
  properties: {
    item: { type: 'string' },
    discrepancies: { type: 'array', items: {
      type: 'object', additionalProperties: false,
      properties: {
        where: { type: 'string', description: 'file:line or verb/skill name' },
        claim: { type: 'string', description: 'what the record claims (or omits)' },
        reality: { type: 'string', description: 'what the binary/tree actually shows' },
        class: { type: 'string', enum: ['false-claim','undocumented-surface','fictional-layout','criterion-violation','stale-count'] },
      }, required: ['where','claim','reality','class'] } },
  }, required: ['item','discrepancies'],
}
// Prompt text comes from the manifest, not from here: the context and both
// direction templates are pinned in manifest.json (with their own promptHash) so
// the prompt cannot drift between runs. The templates carry ${...} placeholders
// as literal text; fill them per checker.
const CTX = input.prompt.context
const fill = (tmpl, subs) =>
  Object.entries(subs).reduce((s, [k, v]) => s.split('${' + k + '}').join(v), tmpl)
const briefFindings = input.briefDocs.map(doc => () => agent(
  `${CTX}\n\n${fill(input.prompt.directionA, { doc })}`,
  { label: `brief:${doc.split('/').pop()}`, phase: 'CheckBrief', schema: FINDINGS }))
const surfFindings = input.surfaces.map(s => () => agent(
  `${CTX}\n\n${fill(input.prompt.directionB, { 's.name': s.name, 's.kind': s.kind, 's.probe': s.probe })}`,
  { label: `surface:${s.name}`, phase: 'CheckSurface', schema: FINDINGS }))
const all = (await parallel([...briefFindings, ...surfFindings]))
  .filter(Boolean)
log(`${all.length} checkers returned`)
const merged = []
const seen = new Set()
for (const r of all) for (const d of r.discrepancies) {
  const k = `${d.where}|${d.claim.slice(0,60)}`
  if (seen.has(k)) continue
  seen.add(k); merged.push({ item: r.item, ...d })
}
log(`${merged.length} unique discrepancies`)
return { count: merged.length, byClass: merged.reduce((m,d)=>(m[d.class]=(m[d.class]||0)+1,m),{}), discrepancies: merged }
