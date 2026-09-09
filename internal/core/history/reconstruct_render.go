package history

// Rendering the reconstruction.
//
// Markdown, because the artefact's consumer is a model being handed the session
// as context and Markdown is what a model reads without a schema. The structure
// is the argument: a contiguous main thread with inline spawn/join markers, one
// appended section per agent, and a timeline table at the head that carries the
// spans. Section ORDER never asserts anything about time; the table is the only
// statement about it, which is what keeps a reader from inferring that a
// sub-agent's conclusions were available to main-thread turns that ran while it
// was still working.

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// stamp is the timestamp format used everywhere in the artefact: second
// precision UTC, which is enough to order turns and short enough to read.
const stamp = "2006-01-02T15:04:05Z"

// spineHeadTurns and spineTailTurns are how much of a sub-agent survives in
// spine mode: its opening instruction and its closing turn. Those are the two
// things a reader of the spine needs — what it was asked, and what it returned.
const (
	spineHeadTurns = 1
	spineTailTurns = 1
)

// render builds the artefact.
//
// The BODY is rendered first, into its own buffer, because rendering is what
// discovers the elisions — and the completeness block, which the head of the
// document carries, has to be able to report them. A completeness block written
// before the elisions it describes would understate the document in the one
// section a reader consults to find out what it is missing.
func (s *session) render() []byte {
	var body strings.Builder

	var elidedBlocks, elidedBytes, omittedTurns int
	r := &renderer{
		maxBlock: s.opts.MaxBlockBytes,
		onElide:  func(n int) { elidedBlocks++; elidedBytes += n },
	}

	if s.main != nil {
		body.WriteString("\n## Main thread\n\n")
		s.renderThread(&body, r, s.main, false)
	} else {
		body.WriteString("\n## Main thread\n\nNo main-thread record for this session is stored. " +
			"The sections below are the sub-agent transcripts that survived; nothing in this " +
			"document can say where any of them was spawned.\n")
	}

	// Placed agents first, then — under their own heading, last, never
	// interleaved — the ones nothing could place. An agent whose spawn point is
	// unknown sitting among agents whose spawn points are known reads as though
	// its position were meant.
	var placed, unplaced []*thread
	for _, t := range s.subs {
		if t.spawnedAtTurn == 0 {
			unplaced = append(unplaced, t)
			continue
		}
		placed = append(placed, t)
	}
	for _, t := range placed {
		s.renderAgentHeader(&body, t)
		omittedTurns += s.renderThread(&body, r, t, s.opts.Mode == ModeSpine)
	}
	if len(unplaced) > 0 {
		fmt.Fprintf(&body, "\n## Unattributed sub-agents\n\nThe %d agent(s) below belong to this "+
			"session, but nothing stored says where in any thread they were spawned. They are "+
			"listed here rather than placed, because a placement this document cannot support "+
			"would be a guess a reader could not tell from a fact.\n", len(unplaced))
		for _, t := range unplaced {
			s.renderAgentHeader(&body, t)
			omittedTurns += s.renderThread(&body, r, t, s.opts.Mode == ModeSpine)
		}
	}

	s.telemetry.Completeness.ElidedBlocks = elidedBlocks
	s.telemetry.Completeness.ElidedBytes = elidedBytes
	s.telemetry.Completeness.OmittedTurns = omittedTurns
	if elidedBlocks > 0 {
		s.telemetry.Completeness.Notes = append(s.telemetry.Completeness.Notes, fmt.Sprintf(
			"%d tool input(s) or result(s) were truncated in the artefact at %d bytes each (%d bytes elided); the telemetry counts the whole transcript, not the truncated render",
			elidedBlocks, s.opts.MaxBlockBytes, elidedBytes))
	}
	if omittedTurns > 0 {
		s.telemetry.Completeness.Notes = append(s.telemetry.Completeness.Notes, fmt.Sprintf(
			"spine mode omitted %d sub-agent turn(s) from the artefact; run again with --mode full for the whole session",
			omittedTurns))
	}

	var b strings.Builder
	s.renderHeader(&b)
	s.renderCompleteness(&b)
	s.renderTimeline(&b)
	b.WriteString(body.String())
	return []byte(b.String())
}

// renderHeader writes the block that tells a reader what they are holding.
//
// Deliberately no generation timestamp: the same records must reconstruct to
// the same bytes, so two runs can be diffed. The generation time is on the
// telemetry, where a changing value costs nothing.
func (s *session) renderHeader(b *strings.Builder) {
	tel := s.telemetry
	fmt.Fprintf(b, "# Session %s\n\n", s.opts.SessionID)
	b.WriteString("One session, reconstructed from abcd's transcript store: the main thread and " +
		"every sub-agent transcript stored for it, in one self-contained document.\n\n")

	fmt.Fprintf(b, "- reconstruction schema: %d\n", reconstructSchemaVersion)
	fmt.Fprintf(b, "- session: `%s`\n", s.opts.SessionID)
	fmt.Fprintf(b, "- root commit: `%s`\n", s.rootSHA)
	fmt.Fprintf(b, "- mode: %s\n", s.opts.Mode)
	fmt.Fprintf(b, "- main thread: %s\n", presentAbsent(s.main != nil))
	fmt.Fprintf(b, "- agents: %d (%d sub-agent transcript(s) besides the main thread)\n",
		len(s.threads), len(s.subs))
	fmt.Fprintf(b, "- records used: %d of %d found\n",
		tel.Completeness.RecordsUsed, tel.Completeness.RecordsFound)
	fmt.Fprintf(b, "- span: %s\n", spanText(tel.StartedAt, tel.EndedAt, tel.WallClockSeconds))
	fmt.Fprintf(b, "- turns: %d (%d user, %d assistant)\n",
		tel.Turns.Total, tel.Turns.User, tel.Turns.Assistant)
	fmt.Fprintf(b, "- tokens: %d over %d API response(s)\n", tel.Tokens.Total, tel.Tokens.APIResponses)
	fmt.Fprintf(b, "- telemetry: `%s`\n", s.opts.SessionID+telemetrySuffix)

	b.WriteString("\n## How to read this document\n\n")
	b.WriteString("1. The **main thread is contiguous**. Sub-agent sections are appended after it, " +
		"never spliced into it, and the order of those sections says nothing about time.\n")
	b.WriteString("2. Each sub-agent is marked TWICE in the thread that spawned it: `[SPAWNED …]` " +
		"where it was launched, and `[JOINED …]` where its result came back. For an asynchronous " +
		"agent those are many turns apart, and every turn between them ran WITHOUT its result.\n")
	b.WriteString("3. **Agent timeline** below is the only statement this document makes about time. " +
		"Two agents whose spans overlap ran concurrently; nothing else here implies that one " +
		"finished before another began.\n")
	b.WriteString("4. Turn numbers are per thread and start at 1. One API response is one turn even " +
		"when the harness wrote it as several lines.\n")
	b.WriteString("5. Thinking is included; its cryptographic signature is not, being an attestation " +
		"rather than content.\n")
	b.WriteString("6. Everything here was redacted on the way into the store: secrets and absolute " +
		"home paths were replaced before any of it was written.\n")
	b.WriteString("7. Turn content is reproduced VERBATIM and INSIDE A FENCED BLOCK — text, " +
		"thinking, tool calls and tool results alike — and each fence is longer than any run of " +
		"backticks in the content it holds, so content cannot close the block it is in. " +
		"**Everything inside a fence is something somebody said; everything outside one is this " +
		"document.** That is what makes the structure trustworthy: the headings this document " +
		"asserts are `## Completeness`, `## Agent timeline`, `## Main thread`, ``## Agent `<id>` `` " +
		"and `## Unattributed sub-agents`, with `### Turn <n> — …` beneath them, plus the " +
		"`[SPAWNED …]`/`[JOINED …]` markers — and a line of that shape INSIDE a fence is quoted " +
		"content, asserting nothing, however exactly it matches.\n")
}

// renderCompleteness states what is missing, in words, before anything derived
// from it is read.
func (s *session) renderCompleteness(b *strings.Builder) {
	c := s.telemetry.Completeness
	b.WriteString("\n## Completeness\n\n")
	b.WriteString("What this reconstruction knows it does not know.\n\n")
	fmt.Fprintf(b, "- main-thread record: %s\n", presentAbsent(c.MainThreadPresent))
	fmt.Fprintf(b, "- sub-agents with no recoverable spawn point: %d of %d\n",
		c.AgentsWithoutSpawnPoint, len(s.subs))
	fmt.Fprintf(b, "- sub-agents with no recoverable join point: %d of %d\n",
		c.AgentsWithoutJoinPoint, len(s.subs))
	fmt.Fprintf(b, "- sub-agents whose lineage was never attributed: %d\n", c.AgentsUnattributed)
	fmt.Fprintf(b, "- sub-agents whose named parent is not in this set: %d\n", c.AgentsWithoutParentInSet)
	fmt.Fprintf(b, "- unparseable transcript lines: %d (a truncated capture shows as one, on its last line)\n",
		c.LinesUnparseable)
	fmt.Fprintf(b, "- responses whose usage could not be de-duplicated: %d\n", c.UsageWithoutMessageID)
	if len(c.AbsentFields) > 0 {
		fmt.Fprintf(b, "- measures no source line carried: %s\n", strings.Join(c.AbsentFields, ", "))
	}
	for _, d := range c.DroppedRecords {
		fmt.Fprintf(b, "- record NOT used: `%s` — %s\n", d.Record, d.Reason)
	}
	for _, n := range c.Notes {
		fmt.Fprintf(b, "- %s\n", n)
	}
}

// renderTimeline writes the table that carries concurrency.
func (s *session) renderTimeline(b *strings.Builder) {
	b.WriteString("\n## Agent timeline\n\n")
	if len(s.subs) == 0 {
		b.WriteString("This session spawned no sub-agents that were stored.\n")
		return
	}
	b.WriteString("| agent | type | depth | parent | spawned | started | ended | joined | turns | tokens |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---:|---:|\n")
	for _, t := range s.ordered() {
		fmt.Fprintf(b, "| %s | %s | %d | %s | %s | %s | %s | %s | %d | %d |\n",
			agentLabel(t), orDash(t.record.AgentType), t.record.SpawnDepth,
			parentLabel(t), pointText(t.spawnedIn, t.spawnedAtTurn),
			timeText(t.started), timeText(t.ended), pointText(t.spawnedIn, t.joinedAtTurn),
			t.turnCount.Total, t.tokens.Total)
	}
	b.WriteString("\nA `spawned`/`joined` cell names a turn in the thread that spawned the agent. " +
		"`—` means the point could not be recovered from what is stored, and is never a claim that " +
		"there was none.\n")
}

// renderAgentHeader writes one sub-agent section's provenance block.
func (s *session) renderAgentHeader(b *strings.Builder, t *thread) {
	fmt.Fprintf(b, "\n## Agent `%s`\n\n", t.record.AgentID)
	fmt.Fprintf(b, "- type: %s\n", orDash(t.record.AgentType))
	fmt.Fprintf(b, "- spawn depth: %d\n", t.record.SpawnDepth)
	fmt.Fprintf(b, "- spawned by: %s\n", parentLabel(t))
	if t.spawnedAtTurn > 0 {
		fmt.Fprintf(b, "- spawned at: %s (tool call `%s`, placed by the %s)\n",
			pointText(t.spawnedIn, t.spawnedAtTurn), orDash(t.spawnToolUse), placedByText(t.placedBy))
	} else {
		b.WriteString("- spawned at: NOT RECOVERABLE from what is stored\n")
	}
	if t.joinedAtTurn > 0 {
		fmt.Fprintf(b, "- joined at: %s\n", pointText(t.spawnedIn, t.joinedAtTurn))
		if t.joinedAtTurn > t.spawnedAtTurn+1 {
			fmt.Fprintf(b, "- CONCURRENCY: %d turn(s) of `%s` ran between the spawn and the join, and none of them had this agent's result\n",
				t.joinedAtTurn-t.spawnedAtTurn-1, t.spawnedIn)
		}
	} else {
		b.WriteString("- joined at: NOT RECOVERABLE from what is stored\n")
	}
	fmt.Fprintf(b, "- lineage attribution: %s\n", orDash(t.record.SpawnAttribution))
	fmt.Fprintf(b, "- span: %s\n", spanText(timePtr(t.started), timePtr(t.ended), secondsBetween(t.started, t.ended)))
	fmt.Fprintf(b, "- turns: %d; tokens: %d over %d API response(s)\n",
		t.turnCount.Total, t.tokens.Total, t.tokens.APIResponses)
	fmt.Fprintf(b, "- record: `%s`\n\n", t.recordName)
}

// renderThread writes one thread's turns and returns how many it omitted.
func (s *session) renderThread(b *strings.Builder, r *renderer, t *thread, spine bool) int {
	if t.unreadable != "" {
		fmt.Fprintf(b, "This record could not be read: %s\n", t.unreadable)
		return 0
	}
	if len(t.turns) == 0 {
		b.WriteString("No conversational turns were recovered from this record.\n")
		return 0
	}
	keep := func(i int) bool {
		if !spine {
			return true
		}
		return i < spineHeadTurns || i >= len(t.turns)-spineTailTurns
	}
	omitted := 0
	gapOpen := false
	for i, tn := range t.turns {
		if !keep(i) {
			omitted++
			gapOpen = true
			continue
		}
		if gapOpen {
			fmt.Fprintf(b, "\n> … %d turn(s) omitted in spine mode …\n", omitted)
			gapOpen = false
		}
		r.renderTurn(b, tn)
		s.renderMarkers(b, t, tn.index)
	}
	if gapOpen {
		fmt.Fprintf(b, "\n> … %d turn(s) omitted in spine mode …\n", omitted)
	}
	return omitted
}

// renderMarkers writes the spawn and join markers that belong at this turn of
// this thread. They are the whole reason the sections can be appended: the
// thread keeps saying where each agent entered and where it came back.
func (s *session) renderMarkers(b *strings.Builder, host *thread, idx int) {
	label := host.label()
	for _, sub := range s.subs {
		if sub.spawnedIn != label {
			continue
		}
		if sub.spawnedAtTurn == idx {
			fmt.Fprintf(b, "\n> **[SPAWNED** agent `%s` (%s) here — its transcript is in section \"Agent `%s`\". "+
				"Everything below this line up to its JOIN marker ran without its result. **]**\n",
				sub.record.AgentID, orDash(sub.record.AgentType), sub.record.AgentID)
		}
		if sub.joinedAtTurn == idx && sub.joinedAtTurn != sub.spawnedAtTurn {
			fmt.Fprintf(b, "\n> **[JOINED** agent `%s` here — its result reached this thread at this turn. **]**\n",
				sub.record.AgentID)
		} else if sub.joinedAtTurn == idx {
			fmt.Fprintf(b, "\n> **[JOINED** agent `%s` here — spawned and joined in the same turn (synchronous). **]**\n",
				sub.record.AgentID)
		}
	}
}

// renderer renders turns under a per-block byte cap.
type renderer struct {
	maxBlock int
	onElide  func(elided int)
}

// renderTurn writes one turn: its heading and its blocks.
func (r *renderer) renderTurn(b *strings.Builder, t turn) {
	fmt.Fprintf(b, "\n### Turn %d — %s", t.index, t.role)
	if t.model != "" {
		fmt.Fprintf(b, " · %s", t.model)
	}
	if !t.at.IsZero() {
		fmt.Fprintf(b, " · %s", t.at.Format(stamp))
	}
	b.WriteString("\n")
	for _, blk := range t.blocks {
		r.renderBlock(b, blk)
	}
}

func (r *renderer) renderBlock(b *strings.Builder, blk rawBlock) {
	switch blk.Type {
	case "text":
		if strings.TrimSpace(blk.Text) == "" {
			return
		}
		// Fenced for the same reason every other block type is, and it is the
		// reason that matters most here: a text block is the one kind of
		// content ANY participant in the session chose the bytes of, and
		// reproduced raw it can emit this document's own headings and markers
		// byte for byte — a section for an agent that never ran, a turn that
		// was never taken, a JOIN marker asserting a result arrived. Redaction
		// has nothing to say about it: those bytes are content the store
		// correctly kept. Containment is the answer, and the fence is the one
		// this file already trusts everywhere else, so there is a single
		// escaping mechanism to be right about rather than a per-type
		// judgement about which content is dangerous. Prose survives a fence
		// intact for the model this artefact is written for; a heading it
		// cannot tell from the document's own does not.
		b.WriteString("\n")
		writeFenced(b, "", blk.Text)
	case "thinking":
		if strings.TrimSpace(blk.Thinking) == "" {
			return
		}
		b.WriteString("\n*thinking:*\n\n")
		writeFenced(b, "", r.cap(blk.Thinking))
	case "tool_use":
		fmt.Fprintf(b, "\n**tool call** `%s`", orDash(blk.Name))
		if blk.ID != "" {
			fmt.Fprintf(b, " (`%s`)", blk.ID)
		}
		b.WriteString("\n\n")
		writeFenced(b, "json", r.cap(compactJSON(blk.Input)))
	case "tool_result":
		fmt.Fprintf(b, "\n**tool result** for `%s`\n\n", orDash(blk.ToolUseID))
		writeFenced(b, "", r.cap(blockText(blk)))
	case "image":
		b.WriteString("\n*(an image block was here; images are not carried into the artefact)*\n")
	default:
		if txt := strings.TrimSpace(blk.Text); txt != "" {
			fmt.Fprintf(b, "\n*(%s)*\n\n", orDash(blk.Type))
			writeFenced(b, "", r.cap(blk.Text))
		}
	}
}

// cap truncates one rendered block and reports what it removed. The truncation
// is stated in the artefact where it happens AND counted into completeness, so
// a short artefact is never mistaken for a short session.
func (r *renderer) cap(s string) string {
	if r.maxBlock <= 0 || len(s) <= r.maxBlock {
		return s
	}
	elided := len(s) - r.maxBlock
	if r.onElide != nil {
		r.onElide(elided)
	}
	return s[:r.maxBlock] + fmt.Sprintf("\n… [%d bytes elided by the per-block cap]", elided)
}

// writeFenced writes a fenced block whose fence is longer than any backtick run
// inside it, so transcript content that itself contains fences cannot break out
// of the block it is in.
func writeFenced(b *strings.Builder, lang, body string) {
	fence := strings.Repeat("`", longestBacktickRun(body)+1)
	if len(fence) < 3 {
		fence = "```"
	}
	b.WriteString(fence)
	b.WriteString(lang)
	b.WriteString("\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	b.WriteString(fence)
	b.WriteString("\n")
}

func longestBacktickRun(s string) int {
	best, run := 0, 0
	for _, c := range s {
		if c == '`' {
			run++
			if run > best {
				best = run
			}
			continue
		}
		run = 0
	}
	return best
}

// compactJSON renders a tool call's input readably, falling back to the raw
// bytes when it is not the object shape it usually is.
func compactJSON(raw []byte) string {
	if len(raw) == 0 {
		return "{}"
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(out)
}

// ---------------------------------------------------------------------------
// small formatting helpers
// ---------------------------------------------------------------------------

func presentAbsent(ok bool) string {
	if ok {
		return "present"
	}
	return "ABSENT"
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func agentLabel(t *thread) string {
	if t.isMain() {
		return "main thread"
	}
	return "`" + t.record.AgentID + "`"
}

func parentLabel(t *thread) string {
	if t.isMain() {
		return "—"
	}
	if t.record.ParentAgentID != "" {
		return "`" + t.record.ParentAgentID + "`"
	}
	if t.record.SpawnAttribution == "unattributed" || t.record.SpawnAttribution == "" {
		return "unknown"
	}
	return "main thread"
}

// pointText names a turn in the thread an agent was spawned in.
func pointText(in string, idx int) string {
	if idx == 0 || in == "" {
		return "—"
	}
	if in == "main" {
		return fmt.Sprintf("main turn %d", idx)
	}
	return fmt.Sprintf("`%s` turn %d", in, idx)
}

func placedByText(rung string) string {
	switch rung {
	case "record":
		return "record's stored spawn tool call"
	case "transcript":
		return "spawning transcript's own tool result"
	default:
		return "—"
	}
}

func timeText(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format(stamp)
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func secondsBetween(a, b time.Time) float64 {
	if a.IsZero() || b.IsZero() {
		return 0
	}
	return b.Sub(a).Seconds()
}

func spanText(start, end *time.Time, seconds float64) string {
	if start == nil || end == nil {
		return "—"
	}
	return fmt.Sprintf("%s → %s (%s)", start.Format(stamp), end.Format(stamp), durationText(seconds))
}

// durationText renders a wall-clock duration for a human reader.
func durationText(seconds float64) string {
	d := time.Duration(seconds * float64(time.Second))
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}
