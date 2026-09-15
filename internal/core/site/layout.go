package site

// The two precomputed chart arrangements, ported from `build_data.py` in
// `.abcd/development/research/abcdev-site/` — the executable spec.
//
// Both are computed HERE, at build time, and ride in record.json as plain
// numbers: the published pages make no API calls and run no layout engine
// (adr-38, adr-48). Determinism is therefore not a nicety but the contract —
// the same tree must produce the same picture, and CI proves it by building
// twice and diffing.
//
// The script leans on numpy and networkx; Go has neither, so the maths is
// written out. Matching Python's exact float output is explicitly NOT the goal
// (no two float pipelines agree to the last bit anyway); matching its SHAPE and
// being reproducible are, so this port is pinned by a golden test of its own
// output rather than by comparison with the script's.

import (
	"math"
	"math/rand"
	"sort"
)

// The coil's constants, all in the renderer's reference pixels.
const (
	// refScale converts the renderer's desktop bubble radius to the reference
	// scale the coil is packed at.
	refScale = 0.86
	// coilGap is the breathing room between neighbours on the coil.
	coilGap = 4.5
	// inwardDip is how far back toward the centre a bubble may fall relative to
	// its predecessor. A larger dip lets the sequence double back and stop
	// reading as a path; zero would force a strict spiral with visible gaps.
	inwardDip = 0.35
	// layoutSeed is the fixed seed for every random draw in this file. The
	// arrangement is a function of the record, and the seed is what keeps the
	// randomness from making it a function of the day as well.
	layoutSeed = 11
	// springIterations, springK and springWeight are the force layout's
	// parameters as the script passes them.
	springIterations = 300
	springThreshold  = 1e-4
	springKFactor    = 1.6
	springWeight     = 3.0
	// decompressPercentile is the radius the dense middle of an island is
	// normalised against before the square-root easing spreads it out.
	decompressPercentile = 97.0
	// islandDensity is how much of an island's disk its bubbles are allowed to
	// fill. It is what sizes an island from what it holds rather than from a
	// fraction of the stage fixed in advance, which is how the old arrangement
	// came to draw sixty records inside a circle with room for six.
	islandDensity = 0.7
	// rimOuter is where the outermost row of records with no typed
	// cross-reference sits.
	rimOuter = 0.97
	// settleRays is how many rays either side of its own the settling pass will
	// try for a bubble whose own ray is congested, and settleSweep the widest
	// step between two of them. Together they are what lets a crowded region
	// spread sideways into the room beside it rather than push the picture off
	// the stage.
	settleRays  = 24
	settleSweep = math.Pi / 6
	// mentionWeight is what a body mention contributes to a bubble's size,
	// relative to a typed link. A mention is evidence of relevance, not of a
	// declared relationship, and the size difference says so.
	mentionWeight = 0.35
)

// typeOrder is the tie-break between records that share a date: the order the
// record itself reads in, decisions first.
var typeOrder = map[string]int{
	"adr": 0, "principle": 1, "intent": 2, "spec": 3, "issue": 4, "rfc": 5, "phase": 6,
}

const typeOrderDefault = 9

// LayoutNode is what both arrangements need to know about one record.
type LayoutNode struct {
	// Type is the record's store name; it sets the bubble's size class and the
	// same-day tie-break.
	Type string
	// Date is the effective date the record is placed by.
	Date string
	// Num is the id number, the last tie-break.
	Num int
	// Touched is the day the record's file last changed. It places nothing — the
	// arrangements are chronological by when a record was WRITTEN — but it is
	// what the span a timeline draws has to reach.
	Touched string
}

// MonthMark is the first record of a month, in placement order.
type MonthMark struct {
	Month string `json:"month"`
	Node  int    `json:"node"`
}

// Point is one placed bubble centre, in the unit disk.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Arrangements holds both precomputed layouts plus the numbers that describe
// them, one entry per node in the caller's own order.
type Arrangements struct {
	// Coil places every record in date order, winding outward.
	Coil []Point `json:"coil"`
	// Links places connected work in islands and everything else on the rim.
	Links []Point `json:"links"`
	// Degree is each record's weighted connectedness; Radius its bubble size in
	// reference pixels.
	Degree []float64 `json:"degree"`
	Radius []float64 `json:"radius"`
	// Order is the coil's placement order.
	Order []int `json:"order"`
	// Months marks where each month begins along the coil.
	Months []MonthMark `json:"months"`
	// Overlaps is the packing's own sanity check: the number of bubble pairs
	// that intersect, summed over BOTH arrangements. It must be zero, and it is
	// published so a reader can see that it is rather than take the claim on
	// trust.
	Overlaps int `json:"overlaps"`
	// CoilRadius is the packed radius before normalisation to the unit disk.
	CoilRadius float64 `json:"coil_radius"`
	// RefScale is the scale the radii are quoted at.
	RefScale float64 `json:"ref_scale"`
	// Isolated counts records with no link of any kind.
	Isolated int `json:"isolated"`
	// Islands are the sizes of the connected groups of three or more; Pairs and
	// Unlinked the counts of the two smaller shapes.
	Islands  []int `json:"islands"`
	Pairs    int   `json:"pairs"`
	Unlinked int   `json:"unlinked"`
	// DateRange is the first and last effective date placed — when the records
	// were WRITTEN.
	DateRange [2]string `json:"date_range"`
	// Span is DateRange extended to the last day any placed record's file was
	// touched, which is the axis a timeline draws: a decision written in May and
	// amended in August occupies both ends of it, and a range that stopped at the
	// writing dates would cut the chart short of its own newest fact.
	Span [2]string `json:"span"`
}

// ComputeArrangements places every record twice: once in date order along the
// coil, once by its typed links. typed and mentions are index pairs into nodes;
// mentions are excluded from the link arrangement (with them it is a hairball)
// but do count toward a bubble's size.
func ComputeArrangements(nodes []LayoutNode, typed, mentions [][2]int) Arrangements {
	n := len(nodes)
	a := Arrangements{
		Coil:     make([]Point, n),
		Links:    make([]Point, n),
		Degree:   make([]float64, n),
		Radius:   make([]float64, n),
		RefScale: refScale,
		Islands:  []int{},
		Months:   []MonthMark{},
		Order:    []int{},
	}
	if n == 0 {
		return a
	}

	for _, e := range typed {
		a.Degree[e[0]]++
		a.Degree[e[1]]++
	}
	for _, e := range mentions {
		a.Degree[e[0]] += mentionWeight
		a.Degree[e[1]] += mentionWeight
	}
	for i, nd := range nodes {
		base, step := 4.5, 3.2
		if nd.Type == "issue" {
			base, step = 3.0, 1.6
		}
		// The cap is what keeps a handful of hubs from swallowing the chart, and
		// the square root already compresses hard: at 4 it flattened everything
		// past sixteen links into one size, so the record's true hubs read the
		// same as a record with a middling few. At 7 they keep growing to about
		// fifty links, which is where this record's connectedness actually runs
		// out, and the hubs are visible as hubs.
		a.Radius[i] = (base + math.Min(math.Sqrt(a.Degree[i]), 7)*step) * refScale
		if a.Degree[i] == 0 {
			a.Isolated++
		}
	}

	a.Order = placementOrder(nodes)
	// A record can carry no date at all — one written but not yet committed, or
	// one git cannot place. An empty string is the smallest string there is, so
	// a plain minimum would publish a span starting at "" and would seat that
	// record at the centre of a chronological spiral, which is the position that
	// means "first". Dateless records are excluded from the span and placed last.
	for _, i := range a.Order {
		if nodes[i].Date == "" {
			continue
		}
		if a.DateRange[0] == "" || nodes[i].Date < a.DateRange[0] {
			a.DateRange[0] = nodes[i].Date
		}
		if nodes[i].Date > a.DateRange[1] {
			a.DateRange[1] = nodes[i].Date
		}
	}
	a.Span = a.DateRange
	for _, nd := range nodes {
		if nd.Touched > a.Span[1] {
			a.Span[1] = nd.Touched
		}
	}

	a.coil(nodes)
	a.byLinks(nodes, typed)
	a.Overlaps = a.overlapCount()
	a.round()
	return a
}

// overlapCount is the gate the build publishes and the CLI prints: the bubble
// pairs that intersect in EITHER arrangement. It used to be the coil's count
// alone, computed inside the coil packer, so a by-links arrangement could ship
// with hundreds of overlapping bubbles and the gate would report zero.
func (a *Arrangements) overlapCount() int {
	return countOverlaps(a.Coil, a.Radius, a.CoilRadius) +
		countOverlaps(a.Links, a.Radius, a.CoilRadius)
}

// countOverlaps is one arrangement's sanity count: the bubble pairs that
// intersect. Both arrangements publish positions normalised to the unit disk
// while radii are quoted in reference pixels, and the renderer reconciles the
// two by drawing a point at point × coil_radius (site-src/record.js). The count
// has to be taken in that same space: comparing a unit-disk distance against a
// pixel radius reports an overlap for very nearly every pair and says nothing
// about the picture a reader sees.
func countOverlaps(points []Point, radii []float64, scale float64) int {
	overlaps := 0
	for i := range points {
		if i >= len(radii) {
			break
		}
		for j := i + 1; j < len(points) && j < len(radii); j++ {
			d := math.Hypot(points[i].X-points[j].X, points[i].Y-points[j].Y) * scale
			if d < radii[i]+radii[j]-0.01 {
				overlaps++
			}
		}
	}
	return overlaps
}

// round publishes the arrangement at the precision the chart draws it, as the
// script does. It is not cosmetic: the last bits of a transcendental are the one
// place two correct builds on two machines can differ, and this build's promise
// is that the same tree renders the same picture wherever it is rendered.
func (a *Arrangements) round() {
	r4 := func(v float64) float64 { return math.Round(v*1e4) / 1e4 }
	for i := range a.Coil {
		a.Coil[i] = Point{X: r4(a.Coil[i].X), Y: r4(a.Coil[i].Y)}
		a.Links[i] = Point{X: r4(a.Links[i].X), Y: r4(a.Links[i].Y)}
		a.Degree[i] = math.Round(a.Degree[i]*10) / 10
		a.Radius[i] = r4(a.Radius[i])
	}
	a.CoilRadius = math.Round(a.CoilRadius*10) / 10
}

// placementOrder sorts records by effective date, then by store, then by id.
func placementOrder(nodes []LayoutNode) []int {
	order := make([]int, len(nodes))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(x, y int) bool {
		a, b := nodes[order[x]], nodes[order[y]]
		// A record with no date is placed after every dated one rather than
		// before all of them: "we cannot date this" is not "this came first".
		if (a.Date == "") != (b.Date == "") {
			return b.Date == ""
		}
		if a.Date != b.Date {
			return a.Date < b.Date
		}
		if ta, tb := typeRank(a.Type), typeRank(b.Type); ta != tb {
			return ta < tb
		}
		return a.Num < b.Num
	})
	return order
}

func typeRank(t string) int {
	if r, ok := typeOrder[t]; ok {
		return r
	}
	return typeOrderDefault
}

// coil places records one by one in date order. The first sits at the centre;
// each next goes beside the previous one — a little further round, as close to
// the centre as the bubbles already placed allow — so the sequence winds outward
// like a snail shell. No simulation, so the picture is the same on every build.
func (a *Arrangements) coil(nodes []LayoutNode) {
	n := len(nodes)
	pos := make([]Point, n)
	pr := make([]float64, n)
	placed := make([]int, 0, n)
	scratch := make([]interval, 0, n)
	phi := 0.0

	for k, i := range a.Order {
		r := a.Radius[i]
		if k == 0 {
			pos[i] = Point{}
			pr[i] = r
			placed = append(placed, i)
			continue
		}
		prev := placed[len(placed)-1]
		prho := math.Hypot(pos[prev].X, pos[prev].Y)
		need := pr[prev] + r + coilGap
		// The ray must clear the previous bubble: at the centre any direction
		// does, so the first step out takes a quarter turn.
		clear := math.Pi / 2
		if prho >= need {
			clear = math.Asin(need / prho)
		}
		phi += clear
		ux, uy := math.Cos(phi), math.Sin(phi)
		rho := clearOutward(math.Max(0, prho-inwardDip*need), ux, uy, r, placed, pos, pr, scratch)

		pos[i] = Point{X: rho * ux, Y: rho * uy}
		pr[i] = r
		placed = append(placed, i)
	}

	rhoMax := 0.0
	for i := 0; i < n; i++ {
		if d := math.Hypot(pos[i].X, pos[i].Y) + pr[i]; d > rhoMax {
			rhoMax = d
		}
	}
	a.CoilRadius = rhoMax

	seen := map[string]bool{}
	for _, i := range a.Order {
		m := nodes[i].Date
		if len(m) > 7 {
			m = m[:7]
		}
		if m == "" || seen[m] {
			continue
		}
		seen[m] = true
		a.Months = append(a.Months, MonthMark{Month: m, Node: i})
	}

	if rhoMax == 0 {
		rhoMax = 1
	}
	for i := 0; i < n; i++ {
		a.Coil[i] = Point{X: pos[i].X / rhoMax, Y: pos[i].Y / rhoMax}
	}
}

// interval is the span of rho along a ray that one resting bubble forbids.
type interval struct{ lo, hi float64 }

// clearOutward walks a bubble outward along a ray from the centre until it
// clears every bubble already at rest. Each resting bubble forbids an interval
// of rho along the ray; the walk jumps past each interval the current rho falls
// inside and repeats until a pass changes nothing, because a later interval can
// push rho back into an earlier one. It is the coil's packing rule, and the
// by-links arrangement settles under the same one.
//
// The intervals need no order: every jump lands on the far end of an interval
// that covers the current rho, so the walk never crosses an uncovered point
// and comes to rest at the least uncovered rho at or beyond where it started
// whatever order the intervals are visited in. scratch is the caller's
// interval buffer, reused across the rays it fires so the pass allocates once
// per bubble rather than once per ray; it must have room for every placed
// bubble.
func clearOutward(rho, ux, uy, r float64, placed []int, pos []Point, pr []float64, scratch []interval) float64 {
	forbidden := scratch[:0]
	for _, j := range placed {
		rr := pr[j] + r + coilGap
		proj := pos[j].X*ux + pos[j].Y*uy
		perp2 := pos[j].X*pos[j].X + pos[j].Y*pos[j].Y - proj*proj
		if perp2 >= rr*rr {
			continue
		}
		half := math.Sqrt(math.Max(rr*rr-perp2, 0))
		forbidden = append(forbidden, interval{proj - half, proj + half})
	}
	for changed := true; changed; {
		changed = false
		for _, iv := range forbidden {
			if iv.lo <= rho && rho < iv.hi {
				rho = iv.hi
				changed = true
			}
		}
	}
	return rho
}

// settle is the one packing rule the whole by-links arrangement comes to rest
// under, and it is the coil's. Innermost first, each of the named bubbles is
// walked outward along its own ray until it clears every bubble already at rest;
// where its own ray is congested the walk is retried on the rays either side, so
// a crowded region spreads sideways into the room beside it rather than pushing
// the picture off the stage. Every bubble is cleared against every bubble
// already placed, so what comes out has no overlapping pair left in it — the
// invariant the old arrangement could only hope for, since it had no collision
// pass at all.
//
// compact says whether a bubble may move toward the centre as well as away from
// it. A region seeded tighter than its bubbles fit needs to, or it stays a third
// wider than it has to be; an arrangement already laid out does not, or the rim
// falls into the holes in the middle.
func settle(pos []Point, radii []float64, members []int, compact bool) {
	order := append([]int(nil), members...)
	rho := make([]float64, len(pos))
	for _, i := range order {
		rho[i] = math.Hypot(pos[i].X, pos[i].Y)
	}
	sort.SliceStable(order, func(x, y int) bool { return rho[order[x]] < rho[order[y]] })

	placed := make([]int, 0, len(order))
	scratch := make([]interval, 0, len(order))
	for _, i := range order {
		theta := 0.0
		if rho[i] > 0 {
			theta = math.Atan2(pos[i].Y, pos[i].X)
		}
		// A ray a bubble's width away is the next distinct place to try; on a
		// tight ring that is a small step and near the centre a large one, so it
		// is capped.
		step := settleSweep
		if rho[i] > 0 {
			step = math.Min(settleSweep, (2*radii[i]+coilGap)/rho[i])
		}
		from := rho[i]
		if compact {
			// Walking from the centre rather than from where the seed left it
			// lets a bubble drop into a gap an earlier one left behind, which is
			// what keeps a crowded region as dense as the coil instead of a third
			// again as wide.
			from = 0
		}
		best, cost := pos[i], math.Inf(1)
		for k := 0; k <= settleRays; k++ {
			for _, side := range [2]float64{1, -1} {
				th := theta + side*float64(k)*step
				ux, uy := math.Cos(th), math.Sin(th)
				d := clearOutward(from, ux, uy, radii[i], placed, pos, radii, scratch)
				p := Point{X: d * ux, Y: d * uy}
				// Compacting, the ray that leaves the bubble nearest the centre
				// wins, so the region closes over its own gaps; settling, the one
				// that moves it least, so it stays where its region put it.
				score := math.Hypot(p.X-pos[i].X, p.Y-pos[i].Y)
				if compact {
					score = d
				}
				if score < cost {
					best, cost = p, score
				}
				if k == 0 {
					// At k = 0 the ray and its mirror are the same ray.
					break
				}
			}
			// A bubble's width from what it was reaching for — the centre when
			// compacting, its seeded place when settling — is near enough;
			// sweeping further only costs time.
			if cost <= radii[i] {
				break
			}
		}
		pos[i] = best
		placed = append(placed, i)
	}
}

// islandRadius is the radius an island's spring layout has to be drawn at for
// its members to have room: the area its bubbles occupy, at the density the
// arrangement packs an island to, spread over the unit disk the layout
// produced. Sizing an island from what it holds is what stops a fixed fraction
// of the stage being asked to hold whatever the record happens to put in it.
func islandRadius(members []int, radii []float64) float64 {
	area := 0.0
	for _, i := range members {
		area += bubbleArea(radii[i])
	}
	return math.Sqrt(area / (math.Pi * islandDensity))
}

// bubbleArea is the room one bubble takes up, its share of the breathing room
// between neighbours included. It is what the core and the rim divide the stage
// by.
func bubbleArea(r float64) float64 {
	w := r + coilGap/2
	return math.Pi * w * w
}

// spreadOnCircle seats a row of things around a circle, each taking an arc in
// proportion to its own width, and returns the angle of each. A row whose widths
// sum to less than the circumference therefore ends up with more room between
// its members than they need, never less.
func spreadOnCircle(widths []float64) []float64 {
	total := 0.0
	for _, w := range widths {
		total += w
	}
	angles := make([]float64, len(widths))
	if total <= 0 {
		return angles
	}
	run := 0.0
	for q, w := range widths {
		angles[q] = 2*math.Pi*(run+w/2)/total - math.Pi/2
		run += w
	}
	return angles
}

// region is one group placed as a unit: an island or a pair, its members held at
// offsets from its own centre.
type region struct {
	members []int
	local   []Point
	extent  float64
	centre  Point
}

// byLinks arranges records by their typed links alone: connected work forms
// islands, pairs ring the main island, and records with no typed
// cross-reference sit on the rim in date order — so a bubble travels a short,
// readable path when the viewer switches arrangement.
//
// Every region is sized from what it holds rather than from a fraction of the
// stage fixed in advance, and the whole arrangement then comes to rest under the
// coil's own packing rule. The old arrangement did neither: its islands were
// settled by a spring layout with no collision pass at all, and its two rim rows
// mapped each record's arc-width onto a circle whose circumference was smaller
// than the sum of those widths, so it published overlapping positions by
// construction and the chart never came to rest on it.
func (a *Arrangements) byLinks(nodes []LayoutNode, typed [][2]int) {
	n := len(nodes)
	adj := make([][]int, n)
	deg := make([]int, n)
	seenEdge := map[[2]int]bool{}
	for _, e := range typed {
		s, t := e[0], e[1]
		if s == t {
			continue
		}
		key := [2]int{min(s, t), max(s, t)}
		if seenEdge[key] {
			continue
		}
		seenEdge[key] = true
		adj[s] = append(adj[s], t)
		adj[t] = append(adj[t], s)
		deg[s]++
		deg[t]++
	}

	comps := components(adj)
	var big [][]int
	var pairs [][]int
	var iso []int
	for _, c := range comps {
		switch {
		case len(c) >= 3:
			big = append(big, c)
		case len(c) == 2:
			pairs = append(pairs, c)
		default:
			iso = append(iso, c[0])
		}
	}
	for _, c := range big {
		a.Islands = append(a.Islands, len(c))
	}
	a.Pairs = len(pairs)
	a.Unlinked = len(iso)

	// Both arrangements are drawn at one scale: the renderer puts a published
	// point at point × coil_radius (site-src/record.js), and quotes every radius
	// in reference pixels. Packing in that space, against the radii as they are
	// actually drawn, is what makes the arrangement's own overlap count mean
	// anything.
	world := a.CoilRadius
	if world <= 0 {
		world = 1
	}
	pos := make([]Point, n)

	rng := rand.New(rand.NewSource(layoutSeed))
	regions := make([]region, 0, len(big)+len(pairs))
	for _, c := range big {
		p := springLayout(c, adj, deg, rng)
		decompress(p)
		scale := islandRadius(c, a.Radius)
		ext := 0.0
		for q, i := range c {
			p[q] = Point{X: p[q].X * scale, Y: p[q].Y * scale}
			ext = math.Max(ext, math.Hypot(p[q].X, p[q].Y)+a.Radius[i])
		}
		regions = append(regions, region{members: c, local: p, extent: ext})
	}
	for _, c := range pairs {
		// A pair is two bubbles side by side, far enough apart to clear each
		// other — which the fixed offset it used to be placed at was not.
		half := (a.Radius[c[0]] + a.Radius[c[1]] + coilGap) / 2
		regions = append(regions, region{
			members: c,
			local:   []Point{{X: -half}, {X: half}},
			extent:  half + math.Max(a.Radius[c[0]], a.Radius[c[1]]),
		})
	}

	// The largest island holds the middle; every other island and every pair
	// takes an arc of a ring around it, as wide as the region itself is.
	ring := regions
	middle := 0.0
	if len(big) > 0 {
		ring = regions[1:]
		middle = regions[0].extent
	}
	if len(ring) > 0 {
		widths := make([]float64, len(ring))
		widest := 0.0
		total := 0.0
		for k, g := range ring {
			widths[k] = 2*g.extent + coilGap
			total += widths[k]
			widest = math.Max(widest, g.extent)
		}
		rr := math.Max(middle+widest+coilGap, total/(2*math.Pi))
		for k, ang := range spreadOnCircle(widths) {
			ring[k].centre = Point{X: math.Cos(ang) * rr, Y: math.Sin(ang) * rr}
		}
	}

	core := make([]int, 0, n)
	for _, g := range regions {
		for q, i := range g.members {
			pos[i] = Point{X: g.centre.X + g.local[q].X, Y: g.centre.Y + g.local[q].Y}
			core = append(core, i)
		}
	}
	// The core and the rim divide the stage between them in proportion to the
	// bubbles each has to hold. A core seeded larger than its share is drawn back
	// to it, so the rim is never left packing itself into whatever room the core
	// happened to ask for — which is the shape of the fault the old arrangement
	// had, with its rim pinned to two circles the core took no account of.
	coreEdge, coreArea, rimArea := 0.0, 0.0, 0.0
	for _, i := range core {
		coreEdge = math.Max(coreEdge, math.Hypot(pos[i].X, pos[i].Y)+a.Radius[i])
		coreArea += bubbleArea(a.Radius[i])
	}
	for _, i := range iso {
		rimArea += bubbleArea(a.Radius[i])
	}
	if share := coreArea + rimArea; share > 0 {
		budget := rimOuter * world * math.Sqrt(coreArea/share)
		if coreEdge > budget {
			for _, i := range core {
				pos[i] = Point{X: pos[i].X * budget / coreEdge, Y: pos[i].Y * budget / coreEdge}
			}
		}
	}

	// The core comes to rest before the rim is laid, so the rim is laid outside
	// what the core actually cost rather than outside what it asked for.
	settle(pos, a.Radius, core, true)
	coreOuter := 0.0
	for _, i := range core {
		coreOuter = math.Max(coreOuter, math.Hypot(pos[i].X, pos[i].Y)+a.Radius[i])
	}

	sort.SliceStable(iso, func(x, y int) bool {
		p, q := nodes[iso[x]], nodes[iso[y]]
		if (p.Date == "") != (q.Date == "") {
			return q.Date == ""
		}
		if p.Date != q.Date {
			return p.Date < q.Date
		}
		if tp, tq := typeRank(p.Type), typeRank(q.Type); tp != tq {
			return tp < tq
		}
		return p.Num < q.Num
	})
	// The rim: every record with no typed cross-reference at all, wound outward
	// around the core in date order under the very rule the coil places by. The
	// old rim mapped each record's arc-width onto one of two circles fixed in
	// advance, whose circumference was smaller than the sum of those widths, so
	// it was overpacked by construction; a band that winds is packed by what it
	// actually holds.
	placed := append([]int(nil), core...)
	scratch := make([]interval, 0, len(core)+len(iso))
	phi, prev := 0.0, -1
	for _, i := range iso {
		r := a.Radius[i]
		// The core's own edge is the floor: a rim record clears every core bubble
		// on the way past it, but it never winds back inside the core.
		rho := coreOuter
		if prev >= 0 {
			prho := math.Hypot(pos[prev].X, pos[prev].Y)
			need := a.Radius[prev] + r + coilGap
			// The ray must clear the record placed before it.
			clear := math.Pi / 2
			if prho >= need {
				clear = math.Asin(need / prho)
			}
			phi += clear
			rho = math.Max(coreOuter, prho-inwardDip*need)
		}
		ux, uy := math.Cos(phi), math.Sin(phi)
		rho = clearOutward(rho, ux, uy, r, placed, pos, a.Radius, scratch)
		pos[i] = Point{X: rho * ux, Y: rho * uy}
		placed = append(placed, i)
		prev = i
	}

	// The arrangement is now as big as its own content needs and no bigger.
	// Where the stage has room left over it is grown to fill it, which can never
	// bring two bubbles closer together; where the record's links need more room
	// than the coil's disk, the arrangement says so rather than publishing an
	// overlap.
	edge := 0.0
	all := make([]int, n)
	for i := range pos {
		all[i] = i
		edge = math.Max(edge, math.Hypot(pos[i].X, pos[i].Y)+a.Radius[i])
	}
	if grow := rimOuter * world / edge; edge > 0 && grow > 1 {
		for i := range pos {
			pos[i] = Point{X: pos[i].X * grow, Y: pos[i].Y * grow}
		}
	}

	// The last word on the invariant: every bubble cleared against every other,
	// so what is published has no overlapping pair left in it.
	settle(pos, a.Radius, all, false)
	for i := range pos {
		a.Links[i] = Point{X: pos[i].X / world, Y: pos[i].Y / world}
	}
}

// components returns the connected components of adj, largest first and then by
// lowest member, so the biggest island is always the one placed in the middle.
func components(adj [][]int) [][]int {
	seen := make([]bool, len(adj))
	var out [][]int
	for i := range adj {
		if seen[i] {
			continue
		}
		var stack []int
		stack = append(stack, i)
		seen[i] = true
		var c []int
		for len(stack) > 0 {
			v := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			c = append(c, v)
			for _, w := range adj[v] {
				if !seen[w] {
					seen[w] = true
					stack = append(stack, w)
				}
			}
		}
		sort.Ints(c)
		out = append(out, c)
	}
	sort.SliceStable(out, func(x, y int) bool {
		if len(out[x]) != len(out[y]) {
			return len(out[x]) > len(out[y])
		}
		return out[x][0] < out[y][0]
	})
	return out
}

// springLayout is the Fruchterman-Reingold force layout over one island, with
// the script's degree weighting: an edge between two well-connected records
// pulls less hard than one between two lonely ones, so a hub does not drag the
// whole island into itself.
func springLayout(members []int, adj [][]int, deg []int, rng *rand.Rand) []Point {
	n := len(members)
	idx := make(map[int]int, n)
	for q, i := range members {
		idx[i] = q
	}
	w := make([][]float64, n)
	for q := range w {
		w[q] = make([]float64, n)
	}
	for q, i := range members {
		for _, j := range adj[i] {
			r, ok := idx[j]
			if !ok {
				continue
			}
			w[q][r] = 1.0 / math.Sqrt(float64(max(deg[i], 1))*float64(max(deg[j], 1))) * springWeight
		}
	}

	pos := make([]Point, n)
	for q := range pos {
		pos[q] = Point{X: rng.NormFloat64() * 0.3, Y: rng.NormFloat64() * 0.3}
	}

	k := springKFactor / math.Sqrt(float64(n))
	// The initial step is a tenth of the layout's own extent, decayed linearly
	// to nothing over the run: big rearrangements early, fine settling late.
	minX, maxX := pos[0].X, pos[0].X
	minY, maxY := pos[0].Y, pos[0].Y
	for _, p := range pos {
		minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
		minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
	}
	t := math.Max(maxX-minX, maxY-minY) * 0.1
	dt := t / float64(springIterations+1)

	disp := make([]Point, n)
	for it := 0; it < springIterations; it++ {
		for q := range disp {
			disp[q] = Point{}
		}
		for q := 0; q < n; q++ {
			for r := 0; r < n; r++ {
				if q == r {
					continue
				}
				dx, dy := pos[q].X-pos[r].X, pos[q].Y-pos[r].Y
				d := math.Hypot(dx, dy)
				if d < 0.01 {
					d = 0.01
				}
				// Repulsion from every node, attraction along weighted edges.
				f := k*k/(d*d) - w[q][r]*d/k
				disp[q].X += dx * f
				disp[q].Y += dy * f
			}
		}
		moved := 0.0
		for q := 0; q < n; q++ {
			l := math.Hypot(disp[q].X, disp[q].Y)
			if l < 0.01 {
				l = 0.1
			}
			ddx, ddy := disp[q].X*t/l, disp[q].Y*t/l
			pos[q].X += ddx
			pos[q].Y += ddy
			moved += ddx*ddx + ddy*ddy
		}
		t -= dt
		if math.Sqrt(moved)/float64(n) < springThreshold {
			break
		}
	}
	rescale(pos)
	return pos
}

// rescale centres a layout and scales it so its widest axis spans the unit
// interval, matching the script's `scale=1.0`.
func rescale(pos []Point) {
	if len(pos) == 0 {
		return
	}
	var mx, my float64
	for _, p := range pos {
		mx += p.X
		my += p.Y
	}
	mx /= float64(len(pos))
	my /= float64(len(pos))
	lim := 0.0
	for q := range pos {
		pos[q].X -= mx
		pos[q].Y -= my
		lim = math.Max(lim, math.Max(math.Abs(pos[q].X), math.Abs(pos[q].Y)))
	}
	if lim <= 0 {
		return
	}
	for q := range pos {
		pos[q].X /= lim
		pos[q].Y /= lim
	}
}

// decompress spreads an island's dense middle: radii are normalised against the
// 97th percentile (so a lone outlier does not shrink everything else), clamped,
// and eased with a square root, which pushes the crowded centre outward while
// leaving the rim where it is.
func decompress(pos []Point) {
	if len(pos) == 0 {
		return
	}
	var mx, my float64
	for _, p := range pos {
		mx += p.X
		my += p.Y
	}
	mx /= float64(len(pos))
	my /= float64(len(pos))

	rad := make([]float64, len(pos))
	ang := make([]float64, len(pos))
	for q, p := range pos {
		dx, dy := p.X-mx, p.Y-my
		rad[q] = math.Hypot(dx, dy)
		ang[q] = math.Atan2(dy, dx)
	}
	ref := percentile(rad, decompressPercentile)
	if ref <= 0 {
		ref = 1
	}
	for q := range pos {
		r := math.Sqrt(math.Min(rad[q]/ref, 1.0))
		pos[q] = Point{X: math.Cos(ang[q]) * r, Y: math.Sin(ang[q]) * r}
	}
}

// percentile is the linear-interpolation percentile of a sample.
func percentile(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	if len(s) == 1 {
		return s[0]
	}
	pos := p / 100 * float64(len(s)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return s[lo]
	}
	return s[lo] + (s[hi]-s[lo])*(pos-float64(lo))
}
