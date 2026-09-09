package history

// Reconstructing one session: a single readable artefact and one telemetry file.
//
// The artefact has ONE consumer in mind — a model being handed the session as
// context — and the telemetry has one purpose: making human-AI and AI-AI
// interaction measurable across runs. Both constraints show up as design
// decisions here rather than as prose elsewhere.
//
// # Why the sub-agents are appended and not physically nested
//
// The spec calls for each sub-agent's section to be nested at its spawn point.
// Measured against the real corpus that renders a false story. The sub-agents
// whose identifier the spawning transcript records are precisely the
// ASYNCHRONOUS ones, and for those the spawn point and the join point (the turn
// where the result came back) are many turns apart. Reconstructing one real
// session here, a delegate spawned at turn 18 of its spawning thread joined at
// turn 167: 148 turns ran in between. Splicing the whole sub-agent section in at the spawn
// point puts its conclusions in front of main-thread turns that happened while
// it was still running, so a reader who reads top to bottom infers that the
// main thread saw those conclusions before it acted. That inference is exactly
// the one this artefact exists to prevent, and it is worse than no
// reconstruction because it is confident.
//
// So the main thread stays CONTIGUOUS and carries two inline markers per agent
// — spawned here, joined here — the agent sections are appended and keyed by
// agent id, and a timeline table at the head carries each agent's spawn turn,
// start, end and join turn. Overlap is then read off the table instead of being
// asserted by section order. Nesting is preserved as DATA (parent, depth, the
// thread the spawn point is in) rather than as indentation.
//
// # Why there is a spine mode
//
// A single unbounded artefact is not usable for the consumer it is for. The
// largest main-thread record in the store on this machine is 38 MB, and one
// session there has 57 records across it; reconstructing a 55-record one in
// full produced a 7.7 MB artefact. `spine` keeps the main thread whole and reduces each
// sub-agent to what a reader of the spine actually needs — what it was asked
// and what it concluded — with the omission stated in the section. There is
// deliberately no per-agent mode: `abcd history show <agent-id>` already
// resolves an agent id to that one transcript, and a second door onto it would
// be a redundant surface.
//
// # What core does and does not do
//
// Core returns bytes and a structure; it writes nothing and knows no path. The
// artefact names records by BASENAME only — never a store path, never a
// directory — which is what keeps it readable with the store gone.

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// reconstructSchemaVersion stamps both the artefact header and the telemetry
// file. A consumer comparing two runs needs to know it is comparing like with
// like, so the version is on the artefact as well as on the JSON.
const reconstructSchemaVersion = 1

// telemetrySuffix and artefactSuffix are the two filenames a front door writes.
// They live here rather than in the CLI because the artefact's header names its
// own telemetry companion, and a name in two places drifts.
const (
	artefactSuffix  = ".md"
	telemetrySuffix = ".telemetry.json"
)

// ReconstructMode selects how much of each sub-agent is rendered. The main
// thread is whole in both modes; only the branches differ.
type ReconstructMode string

const (
	// ModeFull renders every turn of every agent.
	ModeFull ReconstructMode = "full"
	// ModeSpine renders every turn of the main thread and, for each sub-agent,
	// its opening instruction and its closing turn, with the omission stated.
	ModeSpine ReconstructMode = "spine"
)

// ReconstructOptions carries what reconstruction needs beyond the store key.
//
// The spec gives the signature as Reconstruct(rootSHA, sessionID). It grew an
// options struct for one measured reason: the artefact has a size problem the
// spec does not acknowledge (see the package comment above), and both remedies
// — the mode and the per-block cap — are choices only the caller can make.
type ReconstructOptions struct {
	// SessionID is the session to reconstruct. Required.
	SessionID string
	// Mode is full (default) or spine.
	Mode ReconstructMode
	// MaxBlockBytes caps one rendered tool input or tool result. 0 is
	// unbounded. Whatever is elided is counted into the completeness block, so
	// a reader is never left guessing whether a short artefact is a short
	// session.
	MaxBlockBytes int
}

// Reconstruction is one session rendered: the artefact bytes and the telemetry
// that describes them.
type Reconstruction struct {
	// Artefact is the Markdown document. It is deterministic — the same records
	// reconstruct to the same bytes — which is why no generation timestamp
	// appears in it; that lives on the telemetry.
	Artefact []byte `json:"-"`
	// ArtefactName and TelemetryName are the filenames a front door should use.
	ArtefactName  string `json:"artefact_name"`
	TelemetryName string `json:"telemetry_name"`
	// ArtefactBytes is the artefact's length, so a --json caller can report the
	// size without carrying the document through the envelope.
	ArtefactBytes int       `json:"artefact_bytes"`
	Telemetry     Telemetry `json:"telemetry"`
}

// TokenCounts is one agent's (or one session's) token usage.
//
// APIResponses and UsageLinesSeen are BOTH reported, and the gap between them
// is the point. The harness writes one line per content block and repeats the
// same usage object on every line of one response, so summing lines multiplies
// a response's cost by its block count — measured at 2.44x on one stored
// transcript and 5.28x across ten. Counting one usage per distinct message id
// is the only correct reading, and publishing both counters is what lets a
// consumer verify that this file did that rather than take it on trust
// (iss-2609090723027424).
type TokenCounts struct {
	Input              int64 `json:"input"`
	Output             int64 `json:"output"`
	CacheCreationInput int64 `json:"cache_creation_input"`
	CacheReadInput     int64 `json:"cache_read_input"`
	Total              int64 `json:"total"`
	// APIResponses is the number of DISTINCT responses counted — the
	// denominator of every per-response measure.
	APIResponses int `json:"api_responses"`
	// UsageLinesSeen is how many transcript lines carried a usage object.
	UsageLinesSeen int `json:"usage_lines_seen"`
}

func (t *TokenCounts) add(o TokenCounts) {
	t.Input += o.Input
	t.Output += o.Output
	t.CacheCreationInput += o.CacheCreationInput
	t.CacheReadInput += o.CacheReadInput
	t.Total += o.Total
	t.APIResponses += o.APIResponses
	t.UsageLinesSeen += o.UsageLinesSeen
}

// TurnCounts is the turn tally. A turn is one user message or one API response;
// an assistant response split across several transcript lines is ONE turn, on
// the same de-duplication as the tokens.
type TurnCounts struct {
	User      int `json:"user"`
	Assistant int `json:"assistant"`
	Total     int `json:"total"`
}

// AgentTelemetry is one agent's row — the main thread included, as the agent
// with an empty id and depth zero.
type AgentTelemetry struct {
	AgentID          string `json:"agent_id,omitempty"`
	IsMainThread     bool   `json:"is_main_thread"`
	ParentAgentID    string `json:"parent_agent_id,omitempty"`
	AgentType        string `json:"agent_type,omitempty"`
	SpawnDepth       int    `json:"spawn_depth"`
	SpawnAttribution string `json:"spawn_attribution,omitempty"`
	LineageSource    string `json:"lineage_source,omitempty"`
	AdoptedProject   string `json:"adopted_project,omitempty"`
	// Record is the record's BASENAME. Never a path: the artefact and its
	// telemetry must both survive the store being gone.
	Record string `json:"record"`

	StartedAt        *time.Time `json:"started_at,omitempty"`
	EndedAt          *time.Time `json:"ended_at,omitempty"`
	WallClockSeconds float64    `json:"wall_clock_seconds"`

	Turns     TurnCounts     `json:"turns"`
	Tokens    TokenCounts    `json:"tokens"`
	ToolCalls map[string]int `json:"tool_calls"`
	Models    []string       `json:"models"`

	// SpawnedIn names the thread the spawn point was found in — "main" or a
	// parent agent id — and SpawnedAtTurn / JoinedAtTurn are 1-based turn
	// indices in THAT thread. Zero means the point was not recovered, which is
	// reported in completeness rather than guessed.
	SpawnedIn      string `json:"spawned_in,omitempty"`
	SpawnedAtTurn  int    `json:"spawned_at_turn,omitempty"`
	JoinedAtTurn   int    `json:"joined_at_turn,omitempty"`
	SpawnToolUseID string `json:"spawn_tool_use_id,omitempty"`
	// PlacedBy names the rung that placed this agent: record (the stored
	// spawn_tool_use_id), transcript (the spawning transcript's own tool
	// result), or none.
	PlacedBy string `json:"placed_by,omitempty"`

	Lines            int `json:"lines"`
	LinesUnparseable int `json:"lines_unparseable"`
}

// DroppedRecord names a record this reconstruction did NOT use, and why. The
// store can hold several records for one (session, agent) — supersession
// narrows that but does not close it, because a capture through a different
// door writes a fresh record rather than superseding one. Reconstruction picks
// one and says which, here and in the artefact; a silent pick would make two
// runs over the same store disagree with nothing to explain it.
type DroppedRecord struct {
	Record  string `json:"record"`
	AgentID string `json:"agent_id,omitempty"`
	Bytes   int    `json:"bytes"`
	Reason  string `json:"reason"`
}

// Completeness is what this reconstruction knows it does not know.
//
// It is not decoration. A derived measure that cannot say what was missing
// invites the cross-run comparison it cannot support, and the corpus this reads
// is missing things routinely: on this machine 71 sub-agent sets have no parent
// transcript at all.
type Completeness struct {
	// MainThreadPresent is false when no main-thread record exists for this
	// session. Reconstruction still renders — the sub-agents are the material
	// that survived — and every spawn point is then unrecoverable, which is
	// what AgentsWithoutSpawnPoint counts.
	MainThreadPresent bool `json:"main_thread_present"`

	RecordsFound int `json:"records_found"`
	RecordsUsed  int `json:"records_used"`

	// DroppedRecords names every record found and not used.
	DroppedRecords []DroppedRecord `json:"dropped_records,omitempty"`

	AgentsTotal              int `json:"agents_total"`
	AgentsUnattributed       int `json:"agents_unattributed"`
	AgentsWithoutSpawnPoint  int `json:"agents_without_spawn_point"`
	AgentsWithoutJoinPoint   int `json:"agents_without_join_point"`
	AgentsWithoutAgentType   int `json:"agents_without_agent_type"`
	AgentsWithoutParentInSet int `json:"agents_whose_parent_is_not_in_this_set"`

	// LinesUnparseable counts transcript lines that were not JSON. A truncated
	// capture normally shows up as exactly one of these, on the last line.
	LinesUnparseable int `json:"lines_unparseable"`
	// UsageWithoutMessageID counts responses whose usage could not be keyed on
	// a message id and so could not be de-duplicated. A non-zero value is the
	// one condition under which the token totals may be inflated.
	UsageWithoutMessageID int `json:"usage_without_message_id"`

	// AbsentFields names the telemetry inputs no source line carried. It is how
	// "zero tool calls" is told apart from "this harness does not record tool
	// calls".
	AbsentFields []string `json:"absent_fields,omitempty"`

	// ElidedBlocks / ElidedBytes report what the per-block cap removed from the
	// artefact. Telemetry is computed BEFORE any elision, so the counts above
	// describe the whole transcript whatever the artefact shows.
	ElidedBlocks int `json:"elided_blocks"`
	ElidedBytes  int `json:"elided_bytes"`
	// OmittedTurns counts turns spine mode left out of the artefact.
	OmittedTurns int `json:"omitted_turns"`

	// Notes carries anything else a reader must know, in words.
	Notes []string `json:"notes,omitempty"`
}

// Telemetry is the machine-readable companion to the artefact.
type Telemetry struct {
	SchemaVersion int    `json:"schema_version"`
	SessionID     string `json:"session_id"`
	RootCommit    string `json:"root_commit"`
	Mode          string `json:"mode"`
	// GeneratedAt is on the telemetry and NOT on the artefact, which keeps the
	// artefact deterministic: the same records reconstruct to the same bytes.
	GeneratedAt time.Time `json:"generated_at"`

	StartedAt        *time.Time `json:"started_at,omitempty"`
	EndedAt          *time.Time `json:"ended_at,omitempty"`
	WallClockSeconds float64    `json:"wall_clock_seconds"`

	Turns      TurnCounts     `json:"turns"`
	Tokens     TokenCounts    `json:"tokens"`
	ToolCalls  map[string]int `json:"tool_calls"`
	Models     []string       `json:"models"`
	AgentTypes []string       `json:"agent_types"`

	Agents       []AgentTelemetry `json:"agents"`
	Completeness Completeness     `json:"completeness"`
}

// Reconstruct renders one session into a single artefact and its telemetry.
//
// It reads; it never writes and never learns a path. An empty session — no
// record of any kind — is an error, because the caller asked about something
// that is not there. A session whose MAIN THREAD is missing is not: it renders,
// labelled, because that is the majority shape of the sub-agent corpus.
func Reconstruct(rootSHA string, opts ReconstructOptions) (Reconstruction, error) {
	if !rootSHARe.MatchString(rootSHA) {
		return Reconstruction{}, errors.New(rootSHAErrMsg)
	}
	// The session id becomes a filename at the front door, so it is held to the
	// same shape the store holds it to on the way in.
	if !sessionIDRe.MatchString(opts.SessionID) {
		return Reconstruction{}, errors.New("history: sessionID must be non-empty and match [A-Za-z0-9._-]+")
	}
	if opts.Mode == "" {
		opts.Mode = ModeFull
	}
	if opts.Mode != ModeFull && opts.Mode != ModeSpine {
		return Reconstruction{}, fmt.Errorf("history: reconstruct mode %q is not one of full, spine", opts.Mode)
	}
	if opts.MaxBlockBytes < 0 {
		return Reconstruction{}, fmt.Errorf("history: maxBlockBytes must not be negative, got %d", opts.MaxBlockBytes)
	}

	records, err := ListForSession(rootSHA, opts.SessionID)
	if err != nil {
		return Reconstruction{}, err
	}
	if len(records) == 0 {
		return Reconstruction{}, fmt.Errorf("history: no records for session %q under %s", opts.SessionID, rootSHA)
	}

	threads, dropped := loadThreads(records)

	s := &session{
		rootSHA: rootSHA,
		opts:    opts,
		threads: threads,
		dropped: dropped,
		found:   len(records),
	}
	s.order()
	s.place()
	s.measure()
	artefact := s.render()

	return Reconstruction{
		Artefact:      artefact,
		ArtefactName:  opts.SessionID + artefactSuffix,
		TelemetryName: opts.SessionID + telemetrySuffix,
		ArtefactBytes: len(artefact),
		Telemetry:     s.telemetry,
	}, nil
}

// ---------------------------------------------------------------------------
// Loading and choosing records
// ---------------------------------------------------------------------------

// loadThreads reads every record's body and picks ONE per (session, agent).
//
// The pick is longest body, then newest capture, then filename. Longest first
// because that is the store's own notion of more complete — supersession
// replaces a stored record when the new bytes strictly extend it — so
// preferring the newest alone would let a truncated re-capture displace a whole
// transcript. Everything not picked is reported, never dropped silently.
func loadThreads(records []Record) ([]*thread, []DroppedRecord) {
	byAgent := map[string][]*thread{}
	for _, r := range records {
		data, err := fsutil.ReadGuarded(r.Path, maxTranscriptBytes)
		if err != nil {
			// One unreadable record does not sink the session; it is reported.
			byAgent[r.AgentID] = append(byAgent[r.AgentID], &thread{
				record:     r,
				recordName: filepath.Base(r.Path),
				unreadable: err.Error(),
			})
			continue
		}
		rec, body, err := parseRecord(data)
		if err != nil {
			byAgent[r.AgentID] = append(byAgent[r.AgentID], &thread{
				record:     r,
				recordName: filepath.Base(r.Path),
				unreadable: err.Error(),
			})
			continue
		}
		rec.Path = r.Path
		byAgent[r.AgentID] = append(byAgent[r.AgentID], &thread{
			record:     rec,
			recordName: filepath.Base(r.Path),
			body:       body,
		})
	}

	var out []*thread
	var dropped []DroppedRecord
	for agentID, candidates := range byAgent {
		sort.SliceStable(candidates, func(i, j int) bool {
			a, b := candidates[i], candidates[j]
			if a.unreadable != b.unreadable {
				return a.unreadable == "" // readable first
			}
			if len(a.body) != len(b.body) {
				return len(a.body) > len(b.body)
			}
			if !a.record.CapturedAt.Equal(b.record.CapturedAt) {
				return a.record.CapturedAt.After(b.record.CapturedAt)
			}
			return a.recordName > b.recordName
		})
		out = append(out, candidates[0])
		for _, c := range candidates[1:] {
			reason := fmt.Sprintf("a longer or newer record for the same agent was used (%s)", candidates[0].recordName)
			if c.unreadable != "" {
				reason = "unreadable: " + c.unreadable
			}
			dropped = append(dropped, DroppedRecord{
				Record: c.recordName, AgentID: agentID, Bytes: len(c.body), Reason: reason,
			})
		}
		if candidates[0].unreadable != "" {
			dropped = append(dropped, DroppedRecord{
				Record: candidates[0].recordName, AgentID: agentID,
				Reason: "unreadable: " + candidates[0].unreadable,
			})
		}
	}
	sort.SliceStable(dropped, func(i, j int) bool { return dropped[i].Record < dropped[j].Record })
	return out, dropped
}

// thread is one agent's transcript, parsed.
type thread struct {
	record     Record
	recordName string
	body       string
	unreadable string

	turns []turn
	// lines and unparseable describe the source, not the render.
	lines       int
	unparseable int

	tokens    TokenCounts
	toolCalls map[string]int
	models    []string
	turnCount TurnCounts
	noMsgID   int

	started, ended time.Time

	// placement, filled by place().
	spawnedIn     string
	spawnedAtTurn int
	joinedAtTurn  int
	spawnToolUse  string
	placedBy      string
}

// label is how this thread is named in the artefact and in the telemetry.
func (t *thread) label() string {
	if t.record.AgentID == "" {
		return "main"
	}
	return t.record.AgentID
}

func (t *thread) isMain() bool { return t.record.AgentID == "" }

// ---------------------------------------------------------------------------
// Parsing the harness's line-delimited transcript
// ---------------------------------------------------------------------------

// rawLine is the subset of one transcript line this package reads. Everything
// else on the line is ignored rather than refused: the format is the harness's
// and it grows keys without notice, so a strict decode would turn a harness
// upgrade into a reconstruction outage.
type rawLine struct {
	Type      string      `json:"type"`
	Timestamp string      `json:"timestamp"`
	Message   *rawMessage `json:"message"`
}

type rawMessage struct {
	ID      string          `json:"id"`
	Role    string          `json:"role"`
	Model   string          `json:"model"`
	Content json.RawMessage `json:"content"`
	Usage   *rawUsage       `json:"usage"`
}

type rawUsage struct {
	Input              int64 `json:"input_tokens"`
	Output             int64 `json:"output_tokens"`
	CacheCreationInput int64 `json:"cache_creation_input_tokens"`
	CacheReadInput     int64 `json:"cache_read_input_tokens"`
}

type rawBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Thinking  string          `json:"thinking"`
	Name      string          `json:"name"`
	ID        string          `json:"id"`
	ToolUseID string          `json:"tool_use_id"`
	Input     json.RawMessage `json:"input"`
	Content   json.RawMessage `json:"content"`
}

// turn is one user message or one API response. An API response split across
// several transcript lines — which is how the harness writes a multi-block
// response — is ONE turn here, joined on the message id.
type turn struct {
	index     int
	role      string
	at        time.Time
	messageID string
	model     string
	blocks    []rawBlock
	// raw holds the source lines of this turn, used only for locating an agent
	// id mentioned somewhere in a line the block decoder does not reach (a
	// completion notification arrives as an attachment, not as message content).
	raw []string
}

// parse fills the thread from its stored body.
func (t *thread) parse() {
	t.toolCalls = map[string]int{}
	seenUsage := map[string]bool{}
	seenToolUse := map[string]bool{}
	seenModel := map[string]bool{}

	for _, line := range strings.Split(t.body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		t.lines++
		var rl rawLine
		if err := json.Unmarshal([]byte(line), &rl); err != nil {
			t.unparseable++
			continue
		}
		if rl.Type != "user" && rl.Type != "assistant" {
			// Everything else (system notices, queue operations, file history)
			// is transcript plumbing, not conversation. It is not rendered and
			// not counted, but the JOIN marker for an async agent arrives as an
			// attachment on a user line, which IS reached here.
			continue
		}
		if rl.Message == nil {
			continue
		}
		at := parseLineTime(rl.Timestamp)
		t.observe(at)

		blocks := decodeBlocks(rl.Message.Content)

		// An assistant response spread over several lines repeats its message
		// id, so it joins the turn already open rather than starting a new one.
		if rl.Type == "assistant" && rl.Message.ID != "" &&
			len(t.turns) > 0 && t.turns[len(t.turns)-1].messageID == rl.Message.ID {
			cur := &t.turns[len(t.turns)-1]
			cur.blocks = append(cur.blocks, blocks...)
			cur.raw = append(cur.raw, line)
		} else {
			t.turns = append(t.turns, turn{
				index:     len(t.turns) + 1,
				role:      rl.Type,
				at:        at,
				messageID: rl.Message.ID,
				model:     rl.Message.Model,
				blocks:    blocks,
				raw:       []string{line},
			})
			if rl.Type == "user" {
				t.turnCount.User++
			} else {
				t.turnCount.Assistant++
			}
			t.turnCount.Total++
		}

		if m := rl.Message.Model; m != "" && !seenModel[m] {
			seenModel[m] = true
			t.models = append(t.models, m)
		}

		// THE de-duplication. Every line of one response repeats that
		// response's usage object; counting them all multiplies the response's
		// cost by its block count. One usage per distinct message id.
		if u := rl.Message.Usage; u != nil {
			t.tokens.UsageLinesSeen++
			key := rl.Message.ID
			counted := true
			switch {
			case key == "":
				// No id, so nothing to key on. It is counted — dropping it
				// would understate — and the count of these is reported, since
				// they are the only way the total can still be inflated.
				t.noMsgID++
			case seenUsage[key]:
				// A repeat of a response already counted. This is the line the
				// naive sum double-counts.
				counted = false
			default:
				seenUsage[key] = true
			}
			if counted {
				t.tokens.Input += u.Input
				t.tokens.Output += u.Output
				t.tokens.CacheCreationInput += u.CacheCreationInput
				t.tokens.CacheReadInput += u.CacheReadInput
				t.tokens.APIResponses++
			}
		}

		// Tool calls are de-duplicated on the tool call's own id for the same
		// reason: a harness that repeated a response's whole content on every
		// line would otherwise count each call once per line.
		for _, b := range blocks {
			if b.Type != "tool_use" || b.Name == "" {
				continue
			}
			if b.ID != "" {
				if seenToolUse[b.ID] {
					continue
				}
				seenToolUse[b.ID] = true
			}
			t.toolCalls[b.Name]++
		}
	}
	t.tokens.Total = t.tokens.Input + t.tokens.Output +
		t.tokens.CacheCreationInput + t.tokens.CacheReadInput
}

// observe widens the thread's span.
func (t *thread) observe(at time.Time) {
	if at.IsZero() {
		return
	}
	if t.started.IsZero() || at.Before(t.started) {
		t.started = at
	}
	if t.ended.IsZero() || at.After(t.ended) {
		t.ended = at
	}
}

// parseLineTime reads a transcript line's timestamp, tolerating its absence.
func parseLineTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if ts, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return ts.UTC()
	}
	return time.Time{}
}

// decodeBlocks reads a message's content, which the harness writes either as a
// plain string or as an array of typed blocks.
func decodeBlocks(content json.RawMessage) []rawBlock {
	if len(content) == 0 {
		return nil
	}
	var text string
	if err := json.Unmarshal(content, &text); err == nil {
		return []rawBlock{{Type: "text", Text: text}}
	}
	var blocks []rawBlock
	if err := json.Unmarshal(content, &blocks); err != nil {
		return nil
	}
	return blocks
}

// blockText renders a tool_result's content, which has the same
// string-or-array shape as a message's.
func blockText(b rawBlock) string {
	if b.Text != "" {
		return b.Text
	}
	if len(b.Content) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(b.Content, &s); err == nil {
		return s
	}
	// Every text part, not the first: a tool result written as several text
	// blocks would otherwise lose all but one of them, and it is usually the
	// later parts that carry the answer.
	var parts []string
	for _, inner := range decodeBlocks(b.Content) {
		if inner.Text != "" {
			parts = append(parts, inner.Text)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, "\n")
	}
	return string(b.Content)
}

// ---------------------------------------------------------------------------
// The session: ordering, placement, measurement, rendering
// ---------------------------------------------------------------------------

type session struct {
	rootSHA string
	opts    ReconstructOptions
	threads []*thread
	dropped []DroppedRecord
	found   int

	main      *thread
	subs      []*thread
	telemetry Telemetry
}

// order parses every thread and sorts the sub-agents into a stable reading
// order: by depth, then by start time, then by agent id. Start time rather than
// spawn point, because the spawn point is not always recoverable and a section
// order that changes with attribution quality would make two runs over the same
// store disagree.
func (s *session) order() {
	for _, t := range s.threads {
		t.parse()
		if t.isMain() {
			s.main = t
			continue
		}
		s.subs = append(s.subs, t)
	}
	sort.SliceStable(s.subs, func(i, j int) bool {
		a, b := s.subs[i], s.subs[j]
		// A sub-agent record with no depth has no depth RECORDED — depth zero
		// belongs to the main thread alone — so it sorts after everything whose
		// place in the tree is known rather than in front of depth one.
		if au, bu := a.record.SpawnDepth == 0, b.record.SpawnDepth == 0; au != bu {
			return bu
		}
		if a.record.SpawnDepth != b.record.SpawnDepth {
			return a.record.SpawnDepth < b.record.SpawnDepth
		}
		if !a.started.Equal(b.started) {
			return a.started.Before(b.started)
		}
		return a.record.AgentID < b.record.AgentID
	})
}

// place locates each sub-agent's spawn and join points in the thread that
// spawned it — its parent agent's transcript when that is in this set, and the
// main thread otherwise.
//
// Two rungs, most reliable first:
//
//  1. the RECORD's stored spawn_tool_use_id, matched against the tool calls in
//     the spawning thread;
//  2. the spawning TRANSCRIPT itself: the tool result that names this agent's
//     id identifies the call that launched it.
//
// The join point is the turn where the result came back to the spawning thread.
// For a synchronous agent that is the tool result; for an asynchronous one the
// tool result only acknowledges the launch, and the completion arrives many
// turns later, so the LAST-known mention wins over the acknowledgement. Where
// nothing places an agent, both stay zero and completeness counts it. Nothing
// is guessed.
func (s *session) place() {
	byID := map[string]*thread{}
	for _, t := range s.subs {
		byID[t.record.AgentID] = t
	}
	for _, t := range s.subs {
		host := s.main
		if p := t.record.ParentAgentID; p != "" {
			if pt, ok := byID[p]; ok {
				host = pt
			} else {
				// The parent is named but its transcript is not in this set.
				// The main thread is NOT a substitute — placing the agent there
				// would assert a spawn point that did not happen.
				host = nil
			}
		}
		if host == nil {
			continue
		}
		t.spawnedIn = host.label()
		spawnIdx, toolUseID, rung := locateSpawn(host, t)
		if spawnIdx == 0 {
			t.spawnedIn = ""
			continue
		}
		t.spawnedAtTurn = spawnIdx
		t.spawnToolUse = toolUseID
		t.placedBy = rung
		t.joinedAtTurn = locateJoin(host, spawnIdx, toolUseID, t.record.AgentID)
	}
}

// locateSpawn returns the 1-based turn index in host that launched sub, the
// tool call id it was launched by, and which rung answered.
func locateSpawn(host, sub *thread) (int, string, string) {
	if id := sub.record.SpawnToolUseID; id != "" {
		for _, tn := range host.turns {
			for _, b := range tn.blocks {
				if b.Type == "tool_use" && b.ID == id {
					return tn.index, id, "record"
				}
			}
		}
	}
	agentID := sub.record.AgentID
	if agentID == "" {
		return 0, "", ""
	}
	// Rung two. The result of the call that launched an asynchronous agent
	// names that agent's id, so the result identifies the call, and the call's
	// own turn is the spawn point. A synchronous agent's id is not in the
	// spawning transcript at all, which is why rung one exists.
	for _, tn := range host.turns {
		for _, b := range tn.blocks {
			if b.Type != "tool_result" || b.ToolUseID == "" {
				continue
			}
			if !strings.Contains(blockText(b), agentID) {
				continue
			}
			for _, cand := range host.turns {
				for _, cb := range cand.blocks {
					if cb.Type == "tool_use" && cb.ID == b.ToolUseID {
						return cand.index, b.ToolUseID, "transcript"
					}
				}
			}
			// The result is here but the call itself is not (a truncated
			// capture). The result's own turn is the closest true statement.
			return tn.index, b.ToolUseID, "transcript"
		}
	}
	return 0, "", ""
}

// locateJoin returns the 1-based turn index in host at which the sub-agent's
// work came back, or 0 when it cannot be established.
func locateJoin(host *thread, spawnIdx int, toolUseID, agentID string) int {
	join := 0
	for _, tn := range host.turns {
		if tn.index < spawnIdx {
			continue
		}
		if toolUseID != "" {
			for _, b := range tn.blocks {
				if b.Type == "tool_result" && b.ToolUseID == toolUseID && tn.index > join {
					join = tn.index
				}
			}
		}
	}
	if agentID == "" {
		return join
	}
	// An asynchronous completion arrives after the acknowledgement and mentions
	// the agent by id. Scanning the raw line rather than the decoded blocks is
	// deliberate: the notification is carried as an attachment on a user line,
	// which is not message content.
	for _, tn := range host.turns {
		if tn.index <= spawnIdx || tn.index <= join {
			continue
		}
		for _, raw := range tn.raw {
			if strings.Contains(raw, agentID) {
				join = tn.index
				break
			}
		}
	}
	return join
}

// measure fills the telemetry from the parsed threads.
func (s *session) measure() {
	tel := Telemetry{
		SchemaVersion: reconstructSchemaVersion,
		SessionID:     s.opts.SessionID,
		RootCommit:    s.rootSHA,
		Mode:          string(s.opts.Mode),
		GeneratedAt:   time.Now().UTC(),
		ToolCalls:     map[string]int{},
	}
	comp := Completeness{
		MainThreadPresent: s.main != nil,
		RecordsFound:      s.found,
		RecordsUsed:       len(s.threads),
		DroppedRecords:    s.dropped,
		AgentsTotal:       len(s.threads),
	}

	seenModel := map[string]bool{}
	seenType := map[string]bool{}
	var started, ended time.Time

	ordered := s.ordered()
	for _, t := range ordered {
		if !t.started.IsZero() && (started.IsZero() || t.started.Before(started)) {
			started = t.started
		}
		if !t.ended.IsZero() && (ended.IsZero() || t.ended.After(ended)) {
			ended = t.ended
		}
		tel.Turns.User += t.turnCount.User
		tel.Turns.Assistant += t.turnCount.Assistant
		tel.Turns.Total += t.turnCount.Total
		tel.Tokens.add(t.tokens)
		for name, n := range t.toolCalls {
			tel.ToolCalls[name] += n
		}
		for _, m := range t.models {
			if !seenModel[m] {
				seenModel[m] = true
				tel.Models = append(tel.Models, m)
			}
		}
		if ty := t.record.AgentType; ty != "" && !seenType[ty] {
			seenType[ty] = true
			tel.AgentTypes = append(tel.AgentTypes, ty)
		}

		comp.LinesUnparseable += t.unparseable
		comp.UsageWithoutMessageID += t.noMsgID
		if !t.isMain() {
			if t.record.SpawnAttribution == "unattributed" || t.record.SpawnAttribution == "" {
				comp.AgentsUnattributed++
			}
			if t.spawnedAtTurn == 0 {
				comp.AgentsWithoutSpawnPoint++
			}
			if t.joinedAtTurn == 0 {
				comp.AgentsWithoutJoinPoint++
			}
			if t.record.AgentType == "" {
				comp.AgentsWithoutAgentType++
			}
			if p := t.record.ParentAgentID; p != "" && !s.hasAgent(p) {
				comp.AgentsWithoutParentInSet++
			}
		}

		tel.Agents = append(tel.Agents, agentRow(t))
	}
	sort.Strings(tel.Models)
	sort.Strings(tel.AgentTypes)

	if !started.IsZero() {
		st := started
		tel.StartedAt = &st
	}
	if !ended.IsZero() {
		en := ended
		tel.EndedAt = &en
	}
	if !started.IsZero() && !ended.IsZero() {
		tel.WallClockSeconds = ended.Sub(started).Seconds()
	}

	comp.AbsentFields = absentFields(tel)
	if !comp.MainThreadPresent {
		comp.Notes = append(comp.Notes,
			"no main-thread record for this session is stored, so no spawn point or join point can be established for any agent below; the sub-agent transcripts are what survived")
	}
	if comp.UsageWithoutMessageID > 0 {
		comp.Notes = append(comp.Notes, fmt.Sprintf(
			"%d response(s) carried usage with no message id, so their usage could not be de-duplicated; the token totals are an upper bound to that extent",
			comp.UsageWithoutMessageID))
	}
	tel.Completeness = comp
	s.telemetry = tel
}

// ordered is the render and report order: main thread first, then sub-agents.
func (s *session) ordered() []*thread {
	var out []*thread
	if s.main != nil {
		out = append(out, s.main)
	}
	return append(out, s.subs...)
}

func (s *session) hasAgent(id string) bool {
	for _, t := range s.subs {
		if t.record.AgentID == id {
			return true
		}
	}
	return false
}

// agentRow turns one parsed thread into its telemetry row.
func agentRow(t *thread) AgentTelemetry {
	row := AgentTelemetry{
		AgentID:          t.record.AgentID,
		IsMainThread:     t.isMain(),
		ParentAgentID:    t.record.ParentAgentID,
		AgentType:        t.record.AgentType,
		SpawnDepth:       t.record.SpawnDepth,
		SpawnAttribution: t.record.SpawnAttribution,
		LineageSource:    t.record.LineageSource,
		AdoptedProject:   t.record.AdoptedProject,
		Record:           t.recordName,
		Turns:            t.turnCount,
		Tokens:           t.tokens,
		ToolCalls:        t.toolCalls,
		Models:           append([]string(nil), t.models...),
		SpawnedIn:        t.spawnedIn,
		SpawnedAtTurn:    t.spawnedAtTurn,
		JoinedAtTurn:     t.joinedAtTurn,
		SpawnToolUseID:   t.spawnToolUse,
		PlacedBy:         t.placedBy,
		Lines:            t.lines,
		LinesUnparseable: t.unparseable,
	}
	if row.ToolCalls == nil {
		row.ToolCalls = map[string]int{}
	}
	sort.Strings(row.Models)
	if !t.started.IsZero() {
		st := t.started
		row.StartedAt = &st
	}
	if !t.ended.IsZero() {
		en := t.ended
		row.EndedAt = &en
	}
	if !t.started.IsZero() && !t.ended.IsZero() {
		row.WallClockSeconds = t.ended.Sub(t.started).Seconds()
	}
	return row
}

// absentFields names the measures no source line supplied, so a zero can be
// read as "none happened" rather than "this harness does not record it".
func absentFields(tel Telemetry) []string {
	var out []string
	if tel.Tokens.UsageLinesSeen == 0 {
		out = append(out, "tokens")
	}
	if len(tel.ToolCalls) == 0 {
		out = append(out, "tool_calls")
	}
	if len(tel.Models) == 0 {
		out = append(out, "models")
	}
	if len(tel.AgentTypes) == 0 {
		out = append(out, "agent_types")
	}
	if tel.StartedAt == nil {
		out = append(out, "timestamps")
	}
	return out
}
