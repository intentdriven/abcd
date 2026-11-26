package history

// What a sub-agent read, and the line its stop keeps (spc-2611251757228271).
//
// A sub-agent is told at its start where the repository's conventions are and
// which commands print abcd's rules. JudgeSubagent reads the sub-agent's own
// transcript for the two readings, for the stop hook and for the guard's hold
// alike, so the two cannot disagree about what a reading is. Naming is the
// test: a call that names the conventions file counts, whatever it did with
// it, so what is proven is a skipped reading and never a careful one.
//
// The line the stop keeps goes in a marker beside the stage-miss marker, under
// the same lock and atomic write, because both hooks' stderr reaches only a
// debug log: the next session start counts it and `abcd history staged` lists
// it.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// SubagentAsk is what a sub-agent's transcript is read for.
type SubagentAsk struct {
	// File is the conventions file's name, "" where there is none.
	File string
	// RunDir and BriefFile are where a build run keeps its lanes and the name
	// of a lane's brief, which carries the conventions whole. The caller names
	// them, since the loop imports this package.
	RunDir, BriefFile string
	// IsWrite reports whether a call writes: the tool's name, and the command
	// of a shell call ("" otherwise). A call that writes is never a reading.
	IsWrite func(tool, command string) bool
}

// SubagentReading is what the transcript holds: a call that names the
// conventions file, one that names a lane's brief, one that runs the rules
// command, and any call that writes.
type SubagentReading struct {
	ReadFile  bool
	ReadBrief bool
	RanRules  bool
	Wrote     bool
}

// subagentShellTool is the host's shell tool, whose command is read as a
// shell line.
const subagentShellTool = "Bash"

// assistantTypeKeys are the two spellings of the key a line must hold to be
// worth decoding; the decoded line's own type then decides.
var assistantTypeKeys = [][]byte{[]byte(`"type":"assistant"`), []byte(`"type": "assistant"`)}

// JudgeSubagent reads the tool calls a sub-agent made: the tool_use items of
// the transcript's assistant lines. No other line is read, since the host
// copies calls into other lines and names the conventions file in an
// attachment whenever it loaded the file itself. A line that does not decode
// is passed over.
func JudgeSubagent(transcript []byte, ask SubagentAsk) SubagentReading {
	var got SubagentReading
	brief := briefPath(ask.RunDir, ask.BriefFile)
	for line := range bytes.SplitSeq(transcript, []byte("\n")) {
		if !bytes.Contains(line, assistantTypeKeys[0]) && !bytes.Contains(line, assistantTypeKeys[1]) {
			continue
		}
		var rec struct {
			Type    string `json:"type"`
			Message struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &rec) != nil || rec.Type != "assistant" {
			continue
		}
		var items []json.RawMessage
		if json.Unmarshal(rec.Message.Content, &items) != nil {
			continue
		}
		for _, raw := range items {
			var call struct {
				Type  string `json:"type"`
				Name  string `json:"name"`
				Input any    `json:"input"`
			}
			if json.Unmarshal(raw, &call) != nil || call.Type != "tool_use" {
				continue
			}
			got.take(call.Name, call.Input, ask, brief)
		}
	}
	return got
}

// take adds one call to the reading.
func (r *SubagentReading) take(tool string, input any, ask SubagentAsk, brief *regexp.Regexp) {
	command := ""
	if strings.EqualFold(tool, subagentShellTool) {
		if m, ok := input.(map[string]any); ok {
			command, _ = m["command"].(string)
		}
	}
	if command != "" && runsRules(command) {
		r.RanRules = true
	}
	if ask.IsWrite != nil && ask.IsWrite(tool, command) {
		r.Wrote = true
		return
	}
	eachString(input, func(s string) {
		r.ReadFile = r.ReadFile || ask.File != "" && namesFile(s, ask.File)
		r.ReadBrief = r.ReadBrief || brief != nil && brief.MatchString(s)
	})
}

// eachString hands f every string value in a decoded JSON value, at any depth.
func eachString(v any, f func(string)) {
	switch t := v.(type) {
	case string:
		f(t)
	case []any:
		for _, e := range t {
			eachString(e, f)
		}
	case map[string]any:
		for _, e := range t {
			eachString(e, f)
		}
	}
}

// namesFile reports whether s holds name standing alone: with no letter,
// digit, `.`, `_` or `-` on either side of it.
func namesFile(s, name string) bool {
	for from := 0; ; {
		i := strings.Index(s[from:], name)
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(name)
		before, _ := utf8.DecodeLastRuneInString(s[:start])
		after, _ := utf8.DecodeRuneInString(s[end:])
		if (start == 0 || !nameRune(before)) && (end == len(s) || !nameRune(after)) {
			return true
		}
		from = start + 1
	}
}

// nameRune reports whether r continues a file's name.
func nameRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_' || r == '-'
}

// briefPath matches a path that ends in runDir, a run id, a lane id and the
// brief's name, with nothing between and no path element after: a lane's own
// brief, and none of the briefs a run keeps deeper, which do not carry the
// conventions.
func briefPath(runDir, file string) *regexp.Regexp {
	if runDir == "" || file == "" {
		return nil
	}
	return regexp.MustCompile(`(?:^|[^\p{L}\p{N}._-])` + regexp.QuoteMeta(path.Clean(runDir)) +
		`/run-[0-9]+/lane-[0-9]+/` + regexp.QuoteMeta(file) + `(?:$|[^\p{L}\p{N}._/-])`)
}

// runsRules reports whether a shell line runs the rules command: a word whose
// last path element is abcd or abcd.sh, followed by the word rules in the same
// command.
func runsRules(command string) bool {
	prev := ""
	for _, w := range shellWords(command) {
		if w == "rules" && (path.Base(prev) == "abcd" || path.Base(prev) == "abcd.sh") {
			return true
		}
		prev = w
	}
	return false
}

// shellWords splits a shell line into its words, with single quotes, double
// quotes and backslashes honoured, and an empty word where one command ends
// and another can begin, so a word is never taken as following one of another
// command. It reads no expansion and no comment: it is a reading for two
// words, not a shell.
func shellWords(line string) []string {
	var words []string
	var b strings.Builder
	open := false
	end := func() {
		if open {
			words = append(words, b.String())
			b.Reset()
			open = false
		}
	}
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r':
			end()
		case strings.IndexByte("\n;&|()<>`", c) >= 0:
			end()
			words = append(words, "")
		case c == '\'':
			open = true
			j := strings.IndexByte(line[i+1:], '\'')
			if j < 0 {
				b.WriteString(line[i+1:])
				i = len(line)
				break
			}
			b.WriteString(line[i+1 : i+1+j])
			i += j + 1
		case c == '"':
			open = true
			for i++; i < len(line) && line[i] != '"'; i++ {
				if line[i] == '\\' && i+1 < len(line) && strings.IndexByte("\"\\$`", line[i+1]) >= 0 {
					i++
				}
				b.WriteByte(line[i])
			}
		case c == '\\':
			if i+1 < len(line) && line[i+1] == '\n' {
				i++ // a continued line
				break
			}
			open = true
			if i+1 < len(line) {
				i++
				b.WriteByte(line[i])
			}
		default:
			open = true
			b.WriteByte(c)
		}
	}
	end()
	return words
}

// subagentReadingsFilename is the marker the stop's lines are kept in, beside
// the staging directory and the stage-miss marker.
const subagentReadingsFilename = "subagent-readings.json"

// subagentLinesKept is how many lines the marker holds; every line ever kept
// is counted.
const subagentLinesKept = 50

// subagentLineScalarMax bounds the type and the id a line keeps, so the marker
// stays small whatever the host sent.
const subagentLineScalarMax = 120

// maxSubagentReadingsBytes caps the marker on read. Fifty lines whose every
// byte JSON escapes to six are well under it.
const maxSubagentReadingsBytes = 256 << 10

// SubagentLineLock is the wait for the marker's lock on the start hook's
// failure path, so the hook's own bound and the count together stay under the
// time the manifest gives it. The stop hook waits stageMissLockTimeout, as the
// stage-miss marker does.
const SubagentLineLock = time.Second

// SubagentStopLock is the stop hook's wait for the marker's lock.
const SubagentStopLock = stageMissLockTimeout

// SubagentLineKind is what a kept line says, a closed set.
type SubagentLineKind string

const (
	SubagentFileUnread  SubagentLineKind = "file-unread"
	SubagentRulesNotRun SubagentLineKind = "rules-not-run"
	SubagentSkippedBoth SubagentLineKind = "both"
	SubagentNotJudged   SubagentLineKind = "not-judged"
	SubagentStartFailed SubagentLineKind = "start-failed"
)

// SubagentLineCause is why a line could not be judged, a closed set.
type SubagentLineCause string

const (
	// SubagentNoTranscript: the host's stop named no transcript.
	SubagentNoTranscript SubagentLineCause = "no-transcript"
	// SubagentTranscriptUnread: the transcript could not be read.
	SubagentTranscriptUnread SubagentLineCause = "transcript-unread"
	// SubagentTranscriptTooLarge: the transcript is over the size abcd reads.
	SubagentTranscriptTooLarge SubagentLineCause = "transcript-too-large"
	// SubagentFactUnread: abcd's own note that it was told cannot be read.
	SubagentFactUnread SubagentLineCause = "fact-unread"
	// SubagentConventionsUnknown: the guard could not learn whether the
	// conventions apply, so the sub-agent was neither told nor held.
	SubagentConventionsUnknown SubagentLineCause = "conventions-unknown"
)

// SubagentLineHold is what the guard did about the sub-agent's writes, a
// closed set; "" is nothing.
type SubagentLineHold string

const (
	SubagentHoldHeld      SubagentLineHold = "held"
	SubagentHoldNotJudged SubagentLineHold = "not-judged"
	SubagentHoldBoth      SubagentLineHold = "held-and-not-judged"
)

// SubagentLine is one sub-agent's line.
type SubagentLine struct {
	At        time.Time         `json:"at"`
	AgentType string            `json:"agent_type,omitempty"`
	AgentID   string            `json:"agent_id,omitempty"`
	Kind      SubagentLineKind  `json:"kind"`
	Cause     SubagentLineCause `json:"cause,omitempty"`
	Hold      SubagentLineHold  `json:"hold,omitempty"`
}

// valid reports whether every closed field of the line is in its set.
func (l SubagentLine) valid() bool {
	switch l.Kind {
	case SubagentFileUnread, SubagentRulesNotRun, SubagentSkippedBoth, SubagentStartFailed:
		if l.Cause != "" {
			return false
		}
	case SubagentNotJudged:
		switch l.Cause {
		case SubagentNoTranscript, SubagentTranscriptUnread, SubagentTranscriptTooLarge, SubagentFactUnread, SubagentConventionsUnknown:
		default:
			return false
		}
	default:
		return false
	}
	switch l.Hold {
	case "", SubagentHoldHeld, SubagentHoldNotJudged, SubagentHoldBoth:
		return true
	}
	return false
}

// SubagentReadings is the marker: how many lines were ever kept and how many a
// session start has reported, when the first and the last were kept, and the
// newest of them, oldest first.
type SubagentReadings struct {
	Schema    int            `json:"schema"`
	FirstSeen time.Time      `json:"first_seen"`
	LastSeen  time.Time      `json:"last_seen"`
	Count     int            `json:"count"`
	Reported  int            `json:"reported"`
	Lines     []SubagentLine `json:"lines"`
	// EarlierLostAt is when an unreadable marker was replaced: lines kept
	// before it are not in Count. Zero when none was.
	EarlierLostAt time.Time `json:"earlier_lost_at,omitzero"`
}

// subagentReadingsPath returns the marker's path beside the staging directory.
func subagentReadingsPath(repoRoot, rootSHA string) (string, error) {
	if !gitutil.IsFullSHA(rootSHA) {
		return "", errors.New(rootSHAErrMsg)
	}
	lane, err := laneDir(repoRoot, rootSHA)
	if err != nil {
		return "", err
	}
	return filepath.Join(lane, subagentReadingsFilename), nil
}

// NoteSubagentLine keeps one line, under the marker's own lock, waiting at most
// wait for it, so concurrent stops each count. The type and the id are stored
// bounded and as given, and every reader sanitises them before printing.
func NoteSubagentLine(repoRoot, rootSHA string, line SubagentLine, wait time.Duration) error {
	if !line.valid() {
		return fmt.Errorf("history: a sub-agent's line of kind %q, cause %q and hold %q is not one abcd keeps", line.Kind, line.Cause, line.Hold)
	}
	marker, err := subagentReadingsPath(repoRoot, rootSHA)
	if err != nil {
		return err
	}
	line.AgentType, line.AgentID = clipScalar(line.AgentType), clipScalar(line.AgentID)
	return fsutil.WithFileLock(marker+".lock", wait, func() error {
		now := time.Now().UTC()
		line.At = now
		note, ok, rerr := readSubagentReadings(marker)
		switch {
		case rerr != nil:
			// Replaced, and the replacement says when for as long as it
			// stands, as the stage-miss marker's does.
			note = SubagentReadings{FirstSeen: now, EarlierLostAt: now}
		case !ok:
			note = SubagentReadings{FirstSeen: now}
		}
		note.Schema, note.LastSeen = 1, now
		note.Count++
		note.Lines = append(note.Lines, line)
		if over := len(note.Lines) - subagentLinesKept; over > 0 {
			note.Lines = note.Lines[over:]
		}
		return writeSubagentReadings(marker, note)
	})
}

// clipScalar bounds s to subagentLineScalarMax bytes of valid text.
func clipScalar(s string) string {
	if len(s) > subagentLineScalarMax {
		s = s[:subagentLineScalarMax]
	}
	return strings.ToValidUTF8(s, "")
}

// ReadSubagentLines reads the marker. A missing one is not an error: no line
// was kept.
func ReadSubagentLines(repoRoot, rootSHA string) (SubagentReadings, bool, error) {
	marker, err := subagentReadingsPath(repoRoot, rootSHA)
	if err != nil {
		return SubagentReadings{}, false, err
	}
	return readSubagentReadings(marker)
}

// TakeSubagentLines returns how many lines were kept since the last take and
// records them as reported, under the marker's lock, so each line is counted
// in one session's notice.
func TakeSubagentLines(repoRoot, rootSHA string) (int, error) {
	marker, err := subagentReadingsPath(repoRoot, rootSHA)
	if err != nil {
		return 0, err
	}
	if _, err := os.Lstat(marker); errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	fresh := 0
	err = fsutil.WithFileLock(marker+".lock", stageMissLockTimeout, func() error {
		note, ok, rerr := readSubagentReadings(marker)
		if rerr != nil || !ok {
			return rerr
		}
		if fresh = note.Count - note.Reported; fresh <= 0 {
			fresh = 0
			return nil
		}
		note.Reported = note.Count
		return writeSubagentReadings(marker, note)
	})
	if err != nil {
		return 0, err
	}
	return fresh, nil
}

func writeSubagentReadings(marker string, note SubagentReadings) error {
	data, err := json.Marshal(note)
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(marker, append(data, '\n'), 0o600)
}

func readSubagentReadings(marker string) (SubagentReadings, bool, error) {
	data, err := fsutil.ReadGuarded(marker, maxSubagentReadingsBytes)
	if errors.Is(err, os.ErrNotExist) {
		return SubagentReadings{}, false, nil
	}
	if err != nil {
		return SubagentReadings{}, false, err
	}
	var note SubagentReadings
	if err := json.Unmarshal(data, &note); err != nil {
		return SubagentReadings{}, false, fmt.Errorf("history: the marker of sub-agents' readings does not parse: %v", err)
	}
	return note, true, nil
}
