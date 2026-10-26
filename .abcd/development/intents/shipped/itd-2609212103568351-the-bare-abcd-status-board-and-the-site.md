---
id: itd-2609212103568351
slug: the-bare-abcd-status-board-and-the-site
spec_id: spc-2609212138241908
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609211913453478, itd-2609211116005482, itd-2609201916151817]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-100, itd-2609061543533170]
related_adrs: [adr-2609212115255771]
---

# The status board shows Now, Next and Later, computed from the record

## Press Release

> **`abcd` and the site's Status page show Now, Next and Later, computed from the record and maintained by nobody.**
>
> "The phase documents told me what someone once thought would happen next; this tells me what is next," said a product thinker reading the board after retiring phases. "Now is what is in a lane and what the run would pick; Next is what is ready; Later is the rest. Nobody edits it, so it cannot lie."

## Why This Matters

Phases and milestones were retired on 2026-09-21 (adr-2609212115255771) on the evidence that a hand-maintained sequencing document drifts to editorial membership nobody anchors to. The field's replacement for a dated roadmap is the Now / Next / Later view; abcd already holds its inputs in the lifecycle shelves, the readiness gate and the build's state file, so the view is a render, not a record, and it lands where a reader already looks: the bare `abcd` status board and the site.

## Mechanism

We expect a view computed from state to stay true where the hand-maintained phase documents drifted, because nothing on it can be edited into a lie; shown wrong if a reader finds Now naming an intent that is neither in a lane nor the pick order's head.

## Scope Conditions

None stated.

## What's In Scope

- **Now**: every intent whose lane the build's state file shows as started, with its lane state, plus the head of `build next`'s pick order marked "next up", so Now is never empty while anything is READY.
- **Next**: every planned intent the readiness gate reports READY, in pick order.
- **Later**: planned intents not READY (with the failing check named), then drafts.
- **Two surfaces, one read**: the bare `abcd` board gains the block; the site's Status page renders it from the same function; `--json` carries the three lists.
- **Nothing stored**: no folder, no field; removing the state file empties Now's lane rows and leaves the head.
- **No roadmap**: the word appears on neither surface; the glossary points `roadmap` at this block.

## What's Out of Scope

- A started state as a stored field (it is rendered from the state file).
- The pick itself (itd-2609211116005482).
- The site's other pages (itd-2609061543533170).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. Now / Next / Later is rendered status, never stored (adr-2609212115255771, decision 2).
2. When nothing is being built, Now shows the pick order's head marked "next up".
3. The word roadmap is retired; this block is called status.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** an abcd-managed repository, **when** `abcd` runs bare, **then** the board carries a Now / Next / Later block: Now lists the intents the build's state file shows in a lane, with their lane state, and the pick order's head marked "next up"; Next the planned intents the gate reports READY in pick order; Later the planned intents not READY with the failing check named, then the drafts; each row its id and title.
- **Given** the site is built, **when** its Status page is read, **then** it renders the same block from the same read.
- **Given** the build's state file is absent, **when** the board renders, **then** Now holds only the pick order's head, and nothing else on the board changes.
- **Given** `--json`, **when** the board renders, **then** the payload carries the three lists with ids, titles, lane states and the failing checks.
- **Given** either surface, **when** it is searched for the word roadmap, **then** it is absent, and the glossary's `roadmap` entry names this block as its successor.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-4d55b6f29ab8 -->
Fidelity review — receipt rcp-4d55b6f29ab8 (verifier abcd:intent-auditor claude-opus-5-5).

Provenance: abcd:intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:1f2b5bad7479b952289cf7f46d7c30e18c5784a9830cbdbe61a14e5ca8f583fc
Input attestations: diff:git diff --no-color --no-ext-diff 1ded82939 b03b9a748: the statusBlock lane (567e653b7 e3d5ae48d 4a1f36f61 398764ff5 e38a7a067 dff8cbde4 b03b9a748), merged as fb9aceff3@sha256:acea20f3daa96d86dba54943bb1d3c868c216265ae0ea454d290268ecb3be2bb; diff:git diff --no-color --no-ext-diff 9931244fb 7e705cad8: d7dc4b0d5 (Next and the head in build next's pick order), 73f789a85 (captures iss-2609291803334904), 7e705cad8 (closes spc-2609212138241908, Delivers: itd-2609212103568351); refs prefixed 7e705cad8: are read at this ship commit@sha256:a3f22c2fdf1b9739d0214e66726ac9d29ebe8a6636435792bb13ccefbea5be0d; tree:origin/main 2e03c61e4 (code identical at the audit worktree's HEAD), read for the post-ship state; unprefixed refs point here, and a commit ref quotes that commit's message@-;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 4 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: At ship bare abcd renders a status: heading, then Now, Next and Later from statusblock.Read: Now is each incomplete run's intent with its lane state, then the first READY intent in pick order (intent.PickLess over intent.ReadinessIn) that is not held and in no lane, tagged next up; Next is every READY planned intent in that order; Later is every refused planned intent naming its gating failing checks, then every draft; each row carries its id and title (TestBoardCarriesTheStatusBlock, TestBlockPlacesEveryIntent). Concerns: (1) at ship the head applied only the hold and the lane, not build next's other pre-start checks (open question, claim sections, unshipped blocker, no step left, peers), so it could mark next up an intent the pick excludes; the gap was captured as iss-2609291803334904 in 73f789a85, one commit before the close, and fixed only after ship (fc4644eef; the peers check, on the board alone, by dd7f6a76b under ruling CC1). (2) On main the criterion text no longer describes the board: rulings BV1, BV2 and 2026-10-03, recorded in the Audit Notes, make the text board give Later as a count, list an in-lane intent under Now only and the head under Now alone; and spc-2610031844142274 makes bare abcd default to a product-thinker view of titles without ids or the Now/Next/Later headings, the id-bearing rows moving behind --view facilitator.
  evidence: 7e705cad8:internal/surface/cli/cli.go:336 — "renderBoardStatus(w, board.Status)"
  evidence: 7e705cad8:internal/core/statusblock/statusblock.go:177 — "sort.SliceStable(readies, func(i, j int) bool { return intent.PickLess(readies[i].cand, readies[j].cand) })"
  evidence: 7e705cad8:internal/core/statusblock/statusblock.go:185 — "if head == nil && rd.it.Held == "" && !rd.it.HeldMalformed && !inLane[rd.it.ID] {"
  evidence: 7e705cad8:internal/core/statusblock/statusblock.go:171 — "if !c.OK && !c.Advisory {"
  evidence: 7e705cad8:internal/surface/cli/board_status_test.go:77 — "itd-7 The unlinked one [lane-1: implement (run-2609290000000001)]"
  evidence: 7e705cad8:internal/surface/cli/board_status_test.go:79 — "itd-7 The unlinked one [fails: spec_link, spec_body]"
  evidence: 7e705cad8:internal/core/implement/loop/next.go:139 — "if !chk.OK {"
  evidence: .abcd/work/issues/resolved/iss-2609291803334904-the-status-block-s-next-up-head-can-name.md:19 — "The status block's next-up head can name a READY intent abcd build next would not pick."
  evidence: internal/core/board/board.go:308 — "es = append(es, entry{2, fmt.Sprintf("Later: %d %s", len(b.Later), noun)})"
  evidence: internal/surface/cli/board_view.go:27 — "case "", viewProduct:"
  evidence: .abcd/development/intents/shipped/itd-2609212103568351-the-bare-abcd-status-board-and-the-site.md:78 — "the text board gives Later as a count alone while `--json` and the site's Status page keep its rows (criterion 1)"
- ac-2 — MET_WITH_CONCERNS: At ship the site build calls statusblock.Read with the lane reader its front door hands in, abcd site build and abcd lint site hand in loop.StatusLanes exactly as the bare board does, and the record health page, the site's Status page, opens with the block; TestStatusPageRendersTheBlockFromTheSameRead checks that every row Read returns for the same repository and lane reader is on the built page. Concerns: (1) at ship a managed repository whose site-src/ui.json predated the status.* labels was refused at load, so its site would not build after the upgrade until the labels were added by hand, which adr-2609301720596683 calls breaking for that surface; ruling TG1 fixed it after ship. (2) On main the two surfaces no longer make the same read: under ruling CC1 the board hands statusblock.Read loop.StatusPeers while the site hands nil, so they can mark different heads when another checkout holds the pick's first intent, and the text board gives Later as a count where the site lists its rows (BV1).
  evidence: 7e705cad8:internal/core/site/build.go:471 — "block, err := statusblock.Read(repoRoot, req.Lanes)"
  evidence: 7e705cad8:internal/surface/cli/site.go:66 — "Lanes: loop.StatusLanes,"
  evidence: 7e705cad8:internal/surface/cli/board_status.go:32 — "b, err := statusblock.Read(root, loop.StatusLanes)"
  evidence: 7e705cad8:internal/core/site/health.go:53 — "b.WriteString(e.statusSection())"
  evidence: 7e705cad8:internal/core/site/status_test.go:26 — "want, err := statusblock.Read(f.Root(), lanes)"
  evidence: .abcd/development/decisions/adrs/2609301720596683-abcd-adds-a-missing-site-label-to-an.md:35 — "hand, which is breaking for that surface."
  evidence: internal/surface/cli/board_status.go:32 — "b, err := statusblock.Read(root, loop.StatusLanes, loop.StatusPeers)"
  evidence: internal/core/site/build.go:528 — "block, err := statusblock.Read(repoRoot, req.Lanes, nil)"
- ac-3 — MET_WITH_CONCERNS: At ship Next and Later are read from the shelves and the gate alone, the state file only adds Now's lane rows and steers the head past an intent in a lane, and a nil or empty lane reader reads as an absent state file, so without it Now is the head alone; TestBlockWithoutAStateFileKeepsOnlyTheHead asserts Next and Later are unchanged and TestBoardWithoutAStateFileKeepsOnlyTheHead that every board line but the lane row and Now's count is unchanged. Concern, a signed-off divergence: on main ruling BV2, recorded in the Audit Notes against criteria 1 and 3, drops an in-lane intent from Next and Later, so removing the state file now also returns that intent to the list the gate places it in, and nothing else on the board changing no longer holds as written.
  evidence: 7e705cad8:internal/core/statusblock/statusblock.go:95 — "reads as an absent state file. It writes nothing."
  evidence: 7e705cad8:internal/core/statusblock/statusblock_test.go:145 — "if !reflect.DeepEqual(with.Next, without.Next) || !reflect.DeepEqual(with.Later, without.Later) {"
  evidence: 7e705cad8:internal/surface/cli/board_status_test.go:149 — "// Every line but the lane row and Now's count is unchanged."
  evidence: internal/core/statusblock/statusblock.go:216 — "if inLane[it.ID] {"
  evidence: .abcd/development/intents/shipped/itd-2609212103568351-the-bare-abcd-status-board-and-the-site.md:78 — "so without the state file that intent returns to the list the gate places it in (criteria 1 and 3)"
- ac-4 — MET: The bare board's --json envelope carries a status object, the Block, whose now, next and later are always arrays and whose rows carry id, title, bucket, next_up, a lane object (run, lane, step, awaiting) and failing_checks; TestBoardCarriesTheStatusBlock decodes all three lists with the lane state, the next-up head and Later's id, title and failing checks, and TestBlockJSONCarriesTheThreeLists asserts the lane object and failing_checks; on main the payload still carries all of them, the lane's step key renamed stage.
  evidence: 7e705cad8:internal/surface/cli/board.go:44 — "Status *statusblock.Block `json:"status,omitempty"`"
  evidence: 7e705cad8:internal/core/statusblock/statusblock.go:67 — "Failing []string `json:"failing_checks,omitempty"`"
  evidence: 7e705cad8:internal/surface/cli/board_status_test.go:122 — "if s.Now[0].Lane == nil || s.Now[0].Lane.Step != "implement" || s.Now[0].Lane.Run != "run-2609290000000001" || !s.Now[1].NextUp {"
  evidence: 7e705cad8:internal/surface/cli/board_status_test.go:125 — "if s.Later[0].ID != "itd-7" || s.Later[0].Title != "The unlinked one" || len(s.Later[0].Failing) == 0 {"
  evidence: internal/core/statusblock/statusblock.go:131 — "Stage string `json:"stage"`"
- ac-5 — MET_WITH_CONCERNS: The glossary's roadmap entry is status: superseded and opens by naming the rendered Now / Next / Later status block on the abcd board and the site's Status page as what replaces the roadmap document and the word, and neither surface's own words carry the term: the board test greps the whole bare board and the site test the whole built Status page, both over fixtures, and commands/abcd.md, site-src/ui.json and the seed ui.json say no roadmap at ship. Concern, a narrower guarantee than promised: both surfaces print record-sourced names verbatim and nothing keeps the word out of them, so in this repository the bare board's reviews rows print the folder 2026-07-07-roadmap-consistency (on the default board at ship, under --view facilitator on main); the word is absent only from the text the surfaces author, the fixtures carry no such name, and this repository's built Status page could not be checked without building it.
  evidence: .abcd/development/brief/glossary/core/roadmap.md:18 — "the rendered Now / Next / Later status block on the `abcd` board and the site's Status page replaces the roadmap document and the word."
  evidence: .abcd/development/brief/glossary/core/roadmap.md:7 — "status: superseded"
  evidence: 7e705cad8:internal/surface/cli/board_status_test.go:85 — "if strings.Contains(strings.ToLower(text), "roadmap") {"
  evidence: 7e705cad8:internal/core/site/status_test.go:91 — "strings.Contains(strings.ToLower(page), "roadmap")"
  evidence: 7e705cad8:internal/surface/cli/board_reviews.go:92 — "what := r.Folder"
  evidence: 7e705cad8:.abcd/work/reviews/2026-07-07-roadmap-consistency/00-summary.md:1 — "# Roadmap-Consistency Review - Consolidated Summary"
  evidence: internal/surface/cli/board_reviews.go:100 — "termsafe.Sanitize(what), flag))"

Gap audit:
- honoured:
  - Computed from the record and maintained by nobody: the block is one read over the planned and draft shelves, the readiness gate and the build's state file, and nothing is stored
    evidence: 7e705cad8:internal/core/statusblock/statusblock.go:3 — "The block is rendered status, never stored"
    evidence: 7e705cad8:internal/core/statusblock/statusblock.go:95 — "reads as an absent state file. It writes nothing."
  - Next is what is ready, in build next's pick order, and Later is the rest: each refused planned intent with its failing checks, then the drafts
    evidence: 7e705cad8:internal/core/statusblock/statusblock.go:177 — "intent.PickLess(readies[i].cand, readies[j].cand)"
    evidence: 7e705cad8:internal/core/statusblock/statusblock.go:191 — "b.Later = append(b.Later, notReady...)"
  - Two surfaces, one read, as shipped: the bare board and the site's Status page both call statusblock.Read with the loop's lane reader
    evidence: 7e705cad8:internal/surface/cli/board_status.go:32 — "b, err := statusblock.Read(root, loop.StatusLanes)"
    evidence: 7e705cad8:internal/core/site/build.go:471 — "block, err := statusblock.Read(repoRoot, req.Lanes)"
    evidence: 7e705cad8:internal/surface/cli/site.go:66 — "Lanes: loop.StatusLanes,"
  - Removing the state file empties Now's lane rows and leaves the head
    evidence: 7e705cad8:internal/core/statusblock/statusblock_test.go:145 — "if !reflect.DeepEqual(with.Next, without.Next) || !reflect.DeepEqual(with.Later, without.Later) {"
    evidence: 7e705cad8:internal/surface/cli/board_status_test.go:149 — "// Every line but the lane row and Now's count is unchanged."
  - --json carries the three lists
    evidence: 7e705cad8:internal/surface/cli/board.go:44 — "Status *statusblock.Block `json:"status,omitempty"`"
    evidence: 7e705cad8:internal/core/statusblock/statusblock.go:47 — "Now []Row `json:"now"`"
  - The glossary points roadmap at this block
    evidence: .abcd/development/brief/glossary/core/roadmap.md:18 — "the rendered Now / Next / Later status block on the `abcd` board and the site's Status page replaces the roadmap document and the word."
- diverged:
  - Now is what the run would pick (press release): at ship the next-up head was the first READY intent in pick order not held and in no lane, without build next's other pre-start checks, so it could name an intent build next excludes; captured as iss-2609291803334904 one commit before the close and fixed after ship (fc4644eef, then dd7f6a76b for the peers check, on the board alone)
    evidence: 7e705cad8:internal/core/statusblock/statusblock.go:185 — "if head == nil && rd.it.Held == "" && !rd.it.HeldMalformed && !inLane[rd.it.ID] {"
    evidence: 7e705cad8:internal/core/implement/loop/next.go:139 — "if !chk.OK {"
    evidence: .abcd/work/issues/resolved/iss-2609291803334904-the-status-block-s-next-up-head-can-name.md:19 — "The status block's next-up head can name a READY intent abcd build next would not pick."
  - Now is never empty while anything is READY (What's In Scope): when every READY intent is held, Now carries no head and is empty
    evidence: 7e705cad8:internal/core/statusblock/statusblock_test.go:192 — "// With every READY intent held there is no head, and Now is empty."
    evidence: .abcd/development/intents/shipped/itd-2609212103568351-the-bare-abcd-status-board-and-the-site.md:39 — "so Now is never empty while anything is READY."
  - The read in internal/core/positioning, its head read through the same entry point build next uses (spec Scope 1 and Approach): delivered as a new package, internal/core/statusblock, which re-derives the order from intent.PickLess and intent.ReadinessIn because neither positioning nor the site may import the loop, as 567e653b7 records
    evidence: .abcd/development/specs/closed/spc-2609212138241908-the-bare-abcd-status-board-and-the-site.md:28 — "the pick-order head is read through the same entry point `build next` uses so the two never disagree."
    evidence: commit 567e653b7 — "The spec placed the read in internal/core/positioning; that package is the"
  - No roadmap, the word on neither surface: only the text the surfaces author is free of it; record-sourced names print verbatim, and this repository's bare board prints the review folder 2026-07-07-roadmap-consistency
    evidence: 7e705cad8:internal/surface/cli/board_reviews.go:92 — "what := r.Folder"
    evidence: 7e705cad8:.abcd/work/reviews/2026-07-07-roadmap-consistency/00-summary.md:1 — "# Roadmap-Consistency Review - Consolidated Summary"
  - The Status page shows the block in every managed repository as an additive change: at ship a site-src/ui.json without the status.* labels was refused at load, so an upgraded repository's site build failed until ruling TG1 had abcd add the labels
    evidence: .abcd/development/decisions/adrs/2609301720596683-abcd-adds-a-missing-site-label-to-an.md:35 — "hand, which is breaking for that surface."
    evidence: commit 4a1f36f61 — "A repository whose ui.json predates it is refused at load"
  - Two surfaces, one read, on main after ship: ruling CC1 hands the board's read build next's peers check and the site's none, so their heads can differ
    evidence: internal/surface/cli/board_status.go:32 — "b, err := statusblock.Read(root, loop.StatusLanes, loop.StatusPeers)"
    evidence: internal/core/site/build.go:528 — "block, err := statusblock.Read(repoRoot, req.Lanes, nil)"
    evidence: .abcd/work/decisions/dcn-2610020000000005-ruling-cc1-of-the-person-as-technical.md:6 — "the board pays the peers read so 'next up' always equals the pick"
  - The board's block as the criteria describe it, on main after ship: Later is a count on the text board (BV1), an in-lane intent is under Now only (BV2), the head is under Now alone (2026-10-03), and bare abcd defaults to a product-thinker view without ids or the Now/Next/Later headings (spc-2610031844142274), the id-bearing rows under --view facilitator
    evidence: internal/core/board/board.go:308 — "es = append(es, entry{2, fmt.Sprintf("Later: %d %s", len(b.Later), noun)})"
    evidence: internal/core/statusblock/statusblock.go:343 — "b.Next = slices.DeleteFunc(b.Next, func(r Row) bool { return r.ID == head.ID })"
    evidence: internal/core/board/board.go:29 — "// Product is the product thinker's view, the default: what is being"
    evidence: internal/surface/cli/board_view.go:27 — "case "", viewProduct:"
- missing: (none)
<!-- abcd-review-end receipt=rcp-4d55b6f29ab8 -->

Changed on 2026-09-29 by the product thinker's rulings BV1 and BV2 of that day (decision [dcn-2609290000000007](../../../work/decisions/dcn-2609290000000007-the-product-thinker-answered-twenty-six.md) landing with lane recRulings), recorded as iss-2609292011569133: the text board gives Later as a count alone while `--json` and the site's Status page keep its rows (criterion 1), and an intent in a lane is listed under Now only, never also under Next or Later, so without the state file that intent returns to the list the gate places it in (criteria 1 and 3). The criterion text above stands as shipped; adr-2609292012006845 supersedes adr-2609212115255771 and restates decision 2.

Changed on 2026-09-30 by the product thinker's ruling TG1 of that day, recorded as adr-2609301720596683: a managed repository's `site-src/ui.json` written before this intent's `status.*` labels existed keeps building, because `abcd site setup` and `abcd site build` add each declared label the file lacks with abcd's default words, name it on stderr, and change nothing else in the file. The impact stays `additive`.

Changed on 2026-10-03 by the product thinker's ruling of that day ([dcn-2610030000000002](../../../work/decisions/dcn-2610030000000002-ruling-of-the-product-thinker-asked-by.md), recorded with iss-2610031207397996): the head is listed under Now alone, marked "next up", and Next lists the other READY intents in pick order, so no intent appears under both (criterion 1). The criterion text above stands as shipped.

## Grounds

- pursued: phases are retired today and the record needs a place a person looks to see what is next; we expect the computed block to be read where the phase documents were not; shown wrong if Now is found naming an intent neither in a lane nor next
