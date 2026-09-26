---
id: itd-137
slug: maya-sees-how-every-decision-connects-one-relationship-chart
spec_id: spc-39
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-135, itd-136]
severity: minor
impact: additive
---

# Maya sees how every decision connects — one relationship chart with a date coil and a links-only layout, and a five-lane genealogy of the record over time

## Press Release

> **Maya sees how every decision connects — in one chart, not a hairball.**
> Maya studies how software projects actually evolve, and a typed record
> graph of six hundred entries is exactly the artefact they want to read —
> if only anything rendered it. Now `/record/graph/` draws every record as a
> circle — never a shape code — in one chart with two build-time
> arrangements: a date coil that winds outward from the first record with
> concentric month zones like fare zones on a tube map, and a links-only
> force layout where typed cross-references pull connected work into islands.
> Arrowheads appear only where the relation is directed — `builds_on`,
> `supersedes`, `implements` — and mirrored references are collapsed so each
> distinct link draws once. Tapping a record rings it, fades everything
> unlinked, and opens a card: type and state as pills in GitHub's palette,
> every date of the record on one continuum, a pull-out of linked records
> phrased from the focused record, and back and forward buttons that walk the
> viewing history the way a browser does. Controls keep to the corners; the
> middle belongs to the chart; a "Stand by…" overlay covers the settling
> pass; and "Browse as a list" beneath the stage is the keyboard path and the
> accessible twin. `/record/timeline/` is the genealogy: five lanes —
> releases, decisions, intents, specs, issues — over one axis, crowded days
> fanned into capsules with counts, supersession arcs drawn only where both
> ends exist and dashed stubs ending in × where a target has left the tree.
> On a phone the card becomes a bottom sheet and the chart pans to keep the
> focused record visible — perfect at 390 px, or the element is removed from
> the phone view. "Every arrangement I could dismiss as decoration turned
> out to be a rule," said Maya. "The coil is the capture order; the islands
> are the typed links; the stubs are the retirements. The chart taught me
> the record's own conventions."

## Why This Matters

A typed graph nobody can see might as well be untyped. The two arrangements
are the two honest questions — when did work happen, and what does it build
on — and computing both at build time (deterministic, seeded, shipped in
`record.json`) means every visitor reads the same picture, the page runs no
simulation, and the layout cost is paid once per release rather than per
view. Drawing dashed stubs for retired targets rather than inventing
positions keeps the chart inside the record's own truth: the tree does not
carry those files, and the chart says so.

## Acceptance Criteria

- Given `/record/graph/`, then exactly two arrangements are offered by an
  explicit two-state control — the date coil with month zones and the
  links-only force layout — both precomputed at build time into
  `record.json`, deterministic across rebuilds of the same tree. The coil
  packer and the seeded spring embedding are written in-repo against
  `compose.py`/`build_data.py` as the reference implementation; any
  external layout library is a new dependency under the sign-off gate and
  is not assumed.
- Given the stored typed references, then mirrored pairs (intent `spec_id` ↔
  spec `implements`; `related` recorded in both files) collapse to one drawn
  link each, with arrowheads only on `builds_on`, `supersedes` and
  `implements`; body mentions stay off by default.
- Given a focused record, then the card shows the title, type and state
  pills in the GitHub palette, every date of the record on one continuum
  (frontmatter date, in-tree since, state since, last touched — from git),
  a linked-records pull-out phrased from the focused record, and the GitHub
  link; the last-touched date links the record's commit history on GitHub,
  so an amendment is traceable from the card, not just visible as a date;
  back and forward (and Alt+arrows) walk the viewing history.
- Given a keyboard-only visitor, then the list twin under the stage reaches
  every record and every link the chart shows.
- Given `/record/timeline/`, then five lanes render as one static SVG at
  build time; intents, specs and issues take their dates from one
  `git log --reverse --name-status` pass; a day too crowded to show singly
  becomes a capsule with its count; supersession arcs draw only where both
  ends exist, and a target absent from the tree gets a dashed stub ending in
  ×, never an arc to an invented position.
- Given `prefers-reduced-motion`, then all animation is removed, nothing
  drifts at rest, and the list view is offered in place of the chart.
- Given a 390 px viewport, then the card is a full-width bottom sheet, the
  chart pans so the focused record stays above it, and no route scrolls
  horizontally — verified by the static checks in `abcd site check` plus
  the screenshot audit in CI.

## Open Questions

- Whether the retired-ADR stubs graduate to tombstone files (shared with
  itd-136's baseline question).

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-6a21b21ffdd6 -->
Fidelity review — receipt rcp-6a21b21ffdd6 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:9e5b1e28adec5d88a00daa25145783909087b11e988f74c439370351df07d57c
Input attestations: diff:internal/core/site, site-src and .abcd/site.json at chore/audit-run-a-1 80b44890 (git ls-tree -r; spc-39 closed, itd-137 shipped)@sha256:1f54b59e0c5037699d0a1d37669a2f933f882f10104167f5d5dc56debf6a074d;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: the graph page renders a two-button radiogroup switching data-arr between date and links; both arrangements are computed in layout.go at build time — the coil with MonthMark zones and the spring embedding under the fixed layoutSeed — ported in-repo from the research build_data.py with no external layout dependency, and TestBuildIsDeterministic holds every emitted file, record.json included, byte-equal across two builds of one tree
  evidence: internal/core/site/graphpage.go:63 — "role="radiogroup""
  evidence: internal/core/site/layout.go:3 — "The two precomputed chart arrangements, ported from `build_data.py` in"
  evidence: internal/core/site/layout.go:38 — "layoutSeed = 11"
  evidence: internal/core/site/layout.go:93 — "type MonthMark struct {"
  evidence: internal/core/site/build_test.go:696 — "func TestBuildIsDeterministic(t *testing.T) {"
  evidence: internal/core/site/layout_test.go:44 — "func TestCoilNeverOverlaps(t *testing.T) {"
- ac-2 — MET: relationOf normalises spec_id/intent and supersedes/superseded_by to one direction, symmetricRel treats related and implements as one fact stated twice, and collapseEdges drops the mirrored duplicate; the renderer's DIRECTED set is exactly supersedes, implements and builds_on, and mentions start with useMent false behind an unchecked checkbox
  evidence: internal/core/site/recordjson.go:292 — "func collapseEdges(refs []lint.RecordEdge, index map[string]int) ([]ExportEdge, [] [2]int) {"
  evidence: internal/core/site/recordjson.go:174 — "var symmetricRel = map[string]bool{"related": true, "implements": true}"
  evidence: site-src/record.js:82 — "var DIRECTED = ['supersedes', 'implements', 'builds_on'];"
  evidence: site-src/record.js:167 — "useMent = false"
  evidence: internal/core/site/graphpage.go:101 — "< input type="checkbox" id="gmentions">"
- ac-3 — MET: the card renders type, status and severity pills coloured by the --gh-* tokens, a continuum SVG carrying the frontmatter date, the in-tree and state-since dates and the last-touched date read from the git walk, a pull-out whose relation word is phrased from the focused end, a GitHub link, a history link on the last-touched date, and back/forward buttons plus Alt+ArrowLeft/Right over the viewing history
  evidence: site-src/record.js:698 — "var pills = '< span class="pill type">< i>< /i>' + esc(kindOf(n)) + '< /span>' +"
  evidence: site-src/site.css:29 — "--gh-open:#1a7f37; --gh-done:#8250df;"
  evidence: site-src/record.js:616 — "function continuum(n) {"
  evidence: site-src/record.js:655 — "W_.fTouched + ' ' + fmtDate(n.last)"
  evidence: internal/core/site/dates.go:37 — "Touched string `json:"touched"`"
  evidence: site-src/record.js:729 — "foot += '< a class="gh" href="' + esc(W_.commits + n.path) + '">' + esc(W_.history) +"
  evidence: site-src/record.js:742 — "id="bback""
  evidence: site-src/record.js:781 — "if (!canvas.isConnected || !e.altKey || focus < 0) return;"
- ac-4 — MET: the graph page writes a details.blist under the stage from listTwin, which renders every record and every link the chart shows as markup, and a test holds the twin's relation links
  evidence: internal/core/site/graphpage.go:115 — "< details class="blist">"
  evidence: internal/core/site/graphpage.go:264 — "func (e *explorer) listTwin() string {"
  evidence: internal/core/site/explorer_test.go:167 — "func TestListTwinLinksItsRelations(t *testing.T) {"
- ac-5 — MET: timeline.go draws the releases lane, the adr/intent/spec dot lanes and the issues histogram (plus a principles lane the record grew after the promise) as one build-time SVG; dates come from a single git log --reverse --name-status walk in dates.go; a day over a lane's Cap becomes a capsule with its count; arcs draws supersession only where both ends exist and a dashed ×-stub for a target absent from the tree; a test holds the SVG static and deterministic
  evidence: internal/core/site/timeline.go:58 — "var tlLanes = []tlLane{"
  evidence: internal/core/site/timeline.go:150 — "// The issues lane, as a histogram: 374 dots is not a picture."
  evidence: internal/core/site/dates.go:80 — ""log", "--reverse", "--name-status", "-M", "--diff-merges=first-parent","
  evidence: internal/core/site/timeline.go:468 — "// arcs draws supersession where both ends exist, and a dashed ×-stub where the"
  evidence: internal/core/site/timeline.go:522 — "fill="var(--hazard-ink)">× ` + escapeText(u.To) + `< /text>< /g>\`)"
  evidence: internal/core/site/explorer_test.go:289 — "func TestTimelineIsDeterministicStaticSVG(t *testing.T) {"
- ac-6 — MET: the script reads prefers-reduced-motion at load and, when set, removes the stage and opens the list twin in its place before any animation starts, so nothing drifts at rest; the stylesheet zeroes the remaining transitions and the standby spinner under the same query
  evidence: site-src/record.js:27 — "var reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;"
  evidence: site-src/record.js:110 — "if (reduced) { offerList(); return; }"
  evidence: site-src/site.css:625 — "@media (prefers-reduced-motion:reduce){"
- ac-7 — MET_WITH_CONCERNS: below 700px the card becomes a full-width bottom sheet (left and right 8px, width auto), baseTarget pans a focused record to 40px plus 22% of the stage height so it sits above the sheet, checkMobile runs the static overflow checks over every record page and the CI screenshot audit discovers record routes and fails on horizontal scroll at 390px; the concerns are that the gate verb the criterion names, `abcd site check`, is `abcd lint site` at BASE, and that the sheet geometry and the pan are browser-only script with no test — only the overflow outcome is audited
  evidence: site-src/site.css:614 — ".bcard{left:8px;right:8px;width:auto;bottom:8px;"
  evidence: site-src/record.js:188 — "var y = (narrow && focus >= 0)"
  evidence: internal/core/site/check.go:1435 — "func (c *checker) checkMobile() {"
  evidence: site-src/audit/overflow-audit.js:55 — "// The fixed route families. Record pages are discovered below rather than"
  evidence: Makefile:261 — "go run ./cmd/abcd lint site --out"

Gap audit:
- honoured:
  - two precomputed, seeded, deterministic arrangements with no external layout library
    evidence: internal/core/site/layout.go:38 — "layoutSeed = 11"
  - mirrored pairs collapse and arrowheads are confined to the three directed relations
    evidence: site-src/record.js:82 — "var DIRECTED = ['supersedes', 'implements', 'builds_on'];"
  - the card's date continuum and history navigation
    evidence: site-src/record.js:616 — "function continuum(n) {"
  - the keyboard list twin reaches every record and link
    evidence: internal/core/site/explorer_test.go:167 — "func TestListTwinLinksItsRelations(t *testing.T) {"
  - the static timeline with capsules, arcs and dashed stubs
    evidence: internal/core/site/timeline.go:468 — "// arcs draws supersession where both ends exist, and a dashed ×-stub where the"
  - reduced motion removes the chart and offers the list
    evidence: site-src/record.js:110 — "if (reduced) { offerList(); return; }"
- diverged:
  - the mobile checks run in `abcd site check`
    evidence: Makefile:261 — "go run ./cmd/abcd lint site --out"
  - the timeline renders five lanes
    evidence: internal/core/site/timeline.go:62 — "{Type: "principle", R: 4, PerCol: 5, Cap: 10, Height: 84},"
- missing:
  - a test of the bottom-sheet geometry and the focus pan at 390px
    evidence: site-src/record.js:188 — "var y = (narrow && focus >= 0)"
