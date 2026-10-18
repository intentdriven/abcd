package autonomy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/shellquote"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// DefaultBlockCap is the host's cap on consecutive stop-hook continuations:
// after eight in a row it overrides the next block and ends the turn
// (CLAUDE_CODE_STOP_HOOK_BLOCK_CAP raises it).
const DefaultBlockCap = 8

// MaxWakesUnchanged is how many wakes in a row may release an unchanged set
// of capped items (dcn-2610181035361997).
const MaxWakesUnchanged = 2

// The session states while away (dcn-2610180936577720): working (the stop is
// blocked), waiting (allowed with a qualifying wake), and stopped (the host's
// cap overrode a block, recorded).
const (
	StateWorking = "working"
	StateWaiting = "waiting"
	StateStopped = "stopped"
)

// StopInput is what the Stop hook's payload carries, as the front door maps
// it, and the two values it reads from the machine for the verdict.
type StopInput struct {
	SessionID            string
	StopHookActive       bool
	LastAssistantMessage string
	SessionCrons         []SessionCron
	BackgroundTasks      []BackgroundTask
	// Cap is the host's continuation cap; 0 means DefaultBlockCap. The front
	// door reads CLAUDE_CODE_STOP_HOOK_BLOCK_CAP.
	Cap int
	// Location is the zone the host reads its tasks' schedules in; nil means
	// the machine's local time.
	Location *time.Location
}

// Verdict is the stop check's answer.
type Verdict struct {
	// Block holds the stop, with Reason handed to the agent.
	Block  bool
	Reason string
	// SystemMessage is the line shown to the person on an allow while
	// present with work owed (stopped, flagged).
	SystemMessage string
	// Row is the away row that decided (1 to 7), or 0 while present.
	Row int
	// State is the session's state to save, when Save is set (while away).
	State SessionState
	Save  bool
	// Stop is a stop to record in the away record: the block passing the
	// host's cap, possibly forced by it.
	Stop *Stop
	// HandBacks are the hand-backs the verdict accepted, to record in the
	// away record under its lock, once per branch and head.
	HandBacks []OwedItem
}

// Judge is the stop verdict, pure over the away record, the owed set, the
// host's payload, the session's state and now. rec not holding at now is the
// person present: with owed work the stop is allowed with a line naming it,
// and with none silently. While away the rows of the plan are read in order
// and the first that matches decides:
//
//  1. the last message ends on a line ending in `?`, not yet blocked for
//     this message: block with the reroute's reason;
//  2. owed, the set changed (or the first check): block, naming the next
//     item and the count;
//  3. owed, unchanged, a qualifying wake, and no capped item owed or the wake
//     count below two: allow (waiting; the wake count adds one if a capped
//     item is owed);
//  4. owed, unchanged, a qualifying wake, a capped item owed and the wake
//     count at two: block;
//  5. owed, unchanged, no qualifying wake: block;
//  6. none owed, a qualifying wake: allow (waiting);
//  7. none owed, no qualifying wake: block, taking the next ready item first.
//
// stop_hook_active decides no row; it alone drives the count of consecutive
// blocks (false sets it to one, true adds one), and the block whose count
// passes the cap is recorded as a stop possibly forced by the cap.
func Judge(rec *Record, owed Owed, in StopInput, st SessionState, now time.Time) Verdict {
	away := rec.Holds(now)
	if owed.Incomplete {
		// The check ran out of its time budget: it fails open, allowing the
		// stop with a line naming what was not read, and, while away,
		// recording the stop. The session's state is left as it was.
		v := Verdict{SystemMessage: incompleteLine(owed)}
		if away {
			v.Stop = &Stop{Kind: StopCheckIncomplete, Detail: incompleteDetail(owed)}
		}
		return v
	}
	owed, handBacks := settleHandBacks(rec, owed, away)
	if !away {
		if owed.Empty() {
			return Verdict{}
		}
		return Verdict{SystemMessage: presentLine(owed)}
	}

	next := st
	next.SchemaVersion = SchemaVersion
	if cfp := owed.CappedFingerprint(); cfp != next.WakeFingerprint {
		next.Wakes, next.WakeFingerprint = 0, cfp
	}
	fp := owed.Fingerprint()
	changed := fp != st.Fingerprint
	next.Fingerprint = fp
	deadline := Deadline(owed, now)
	_, look, wake := QualifyingWake(in.SessionCrons, now, deadline, in.Location)

	v := Verdict{Save: true, HandBacks: handBacks}
	qhash := ""
	if endsOnQuestion(in.LastAssistantMessage) {
		qhash = hashText(in.LastAssistantMessage)
	}
	switch {
	case qhash != "" && qhash != st.QuestionHash:
		v.Row, v.Reason = 1, questionReason()
		next.QuestionHash = qhash
		// The owed set is not taken as seen: the next check names it.
		next.Fingerprint = st.Fingerprint
	case !owed.Empty() && changed:
		v.Row, v.Reason = 2, owedReason(rec, owed, "")
	case !owed.Empty() && wake && (!owed.HasCapped() || next.Wakes < MaxWakesUnchanged):
		v.Row = 3
		if owed.HasCapped() {
			next.Wakes++
		}
	case !owed.Empty() && wake:
		v.Row, v.Reason = 4, owedReason(rec, owed,
			"The set has not moved over two wakes, so a wake no longer releases this stop: move the item, or take another.")
	case !owed.Empty():
		v.Row, v.Reason = 5, owedReason(rec, owed,
			fmt.Sprintf("If it cannot move, take another or schedule a look before %s (a cron that fires by then, the host's lateness included).", clock(deadline, in.Location)))
	case wake:
		v.Row = 6
	default:
		v.Row = 7
		v.Reason = fmt.Sprintf("abcd: the person is away and nothing is owed: take the next ready item first (reserve it with `abcd reserve <id>`); only if none can be taken alone, schedule a look before %s (a cron that fires by then, the host's lateness included).",
			clock(deadline, in.Location))
	}

	if v.Reason == "" {
		v.Block = false
		next.State, next.NextLook, next.Blocks = StateWaiting, look, 0
		v.State = next
		return v
	}
	v.Block = true
	next.State, next.NextLook = StateWorking, time.Time{}
	if in.StopHookActive {
		next.Blocks++
	} else {
		next.Blocks = 1
	}
	limit := in.Cap
	if limit <= 0 {
		limit = DefaultBlockCap
	}
	if next.Blocks > limit {
		row := v.Row
		if row == 1 {
			// A question blocked past the cap is a stop like any other:
			// with work owed, or with none and no look scheduled.
			row = 7
			if !owed.Empty() {
				row = 2
			}
		}
		switch row {
		case 2, 4, 5:
			item, _ := owed.Next()
			v.Stop = &Stop{Kind: StopWorkOwed, Detail: describeItem(item), PossiblyForcedByCap: true}
			next.State = StateStopped
		case 7:
			v.Stop = &Stop{Kind: StopNoNextLook, PossiblyForcedByCap: true}
			next.State = StateStopped
		}
	}
	v.State = next
	return v
}

// settleHandBacks applies the hand-back rule at the verdict (F1,
// dcn-2610181035355741): only while away and only under --hold-pushes is an
// item handed back, and then only an unpushed branch's item with its head;
// every other item the reader handed back is put back among the owed items.
func settleHandBacks(rec *Record, owed Owed, away bool) (Owed, []OwedItem) {
	if len(owed.HandedBack) == 0 {
		return owed, nil
	}
	items := append([]OwedItem(nil), owed.Items...)
	var accepted []OwedItem
	for _, h := range owed.HandedBack {
		if away && rec.HoldPushes && h.Kind == OwedUnpushed && h.Branch != "" && h.Head != "" {
			accepted = append(accepted, h)
			continue
		}
		items = append(items, h)
	}
	owed.Items, owed.HandedBack = items, nil
	return owed, accepted
}

// endsOnQuestion reports whether the message's last line, trailing space and
// emphasis marks aside, ends in a question mark.
func endsOnQuestion(msg string) bool {
	return strings.HasSuffix(strings.TrimRight(msg, " \t\r\n*_"), "?")
}

func hashText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:16])
}

func questionReason() string {
	return "abcd: your last message ends on a question, and the person is away, so no one will answer it. Do not ask it.\n" + QuestionRoute()
}

func owedReason(rec *Record, owed Owed, tail string) string {
	item, _ := owed.Next()
	var b strings.Builder
	fmt.Fprintf(&b, "abcd: the person is away and work is owed (%s). Next: %s.", count(len(owed.Items), "item", "items"), describeItem(item))
	if needsPush(item.Kind) {
		b.WriteString(" " + pushWords(rec, item))
	}
	if tail != "" {
		b.WriteString(" " + tail)
	}
	for _, n := range owed.Notes {
		b.WriteString(" " + termsafe.CleanProseLine(n, 256))
	}
	return b.String()
}

// needsPush reports whether an item's next act, once the local work is done,
// is a push (or opening a pull request, the same kind of act).
func needsPush(k OwedKind) bool {
	switch k {
	case OwedUnpushed, OwedUncarried, OwedBehind, OwedFailing:
		return true
	}
	return false
}

// pushWords is the reason's sentence for an item that needs a push
// (dcn-2610181016510153, refined by dcn-2610181035355741). It defers to the
// person's standing instruction and never names the away mode as leave.
func pushWords(rec *Record, item OwedItem) string {
	if rec != nil && rec.HoldPushes {
		target := "<branch>"
		if item.Branch != "" {
			target = clean(item.Branch)
		}
		return fmt.Sprintf("With --hold-pushes the person has said pushes wait: finish the local work (the merge from the base, the preflight), then queue the push naming the branch, as `abcd mode queue %s`.", shellWord("push "+target))
	}
	return "Do the local work, then push if your standing instruction allows; queue only if the person has said pushes wait. A push you may not send stays owed: queue it with `abcd mode queue \"<step>\"` for the person and take another item meanwhile."
}

// shellWord is s as one word of a command the reason suggests: as it is when
// it holds only characters no shell reads specially, else single-quoted
// (shellquote.Single), so a branch name or record id holding `$(…)`, a
// backquote or a quote cannot run anything when the suggestion is pasted.
func shellWord(s string) string {
	if s != "" && strings.IndexFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._/-", r))
	}) < 0 {
		return s
	}
	return shellquote.Single(s)
}

// describeItem names an owed item in plain words: the reader's reason when it
// wrote one, which already names the item, else words built from its fields.
func describeItem(i OwedItem) string {
	if i.Reason != "" {
		return termsafe.CleanProseLine(i.Reason, 256)
	}
	pr := ""
	if i.PR > 0 {
		pr = fmt.Sprintf("pull request #%d", i.PR)
		if i.Branch != "" {
			pr += " (" + clean(i.Branch) + ")"
		}
	}
	var s string
	switch i.Kind {
	case OwedRenewal:
		s = "the reservation on " + clean(i.Record) + " is due for renewal"
	case OwedFailing:
		s = orBranch(pr, i.Branch) + " is failing its checks"
	case OwedBehind:
		s = orBranch(pr, i.Branch) + " is behind its base"
	case OwedStacked:
		s = "branch " + clean(i.Branch) + " is stacked on a change still open and needs work"
	case OwedUnpushed:
		s = "branch " + clean(i.Branch) + " has commits not pushed"
	case OwedUncarried:
		s = "the reservation on " + clean(i.Record) + " has no open pull request carrying its work"
	default:
		s = clean(string(i.Kind)) + " " + clean(i.Branch+i.Record)
	}
	return s
}

func orBranch(pr, branch string) string {
	if pr != "" {
		return pr
	}
	return "branch " + clean(branch)
}

// incompleteDetail is what an incomplete reading's notes say was not read.
func incompleteDetail(owed Owed) string {
	for _, n := range owed.Notes {
		if strings.Contains(n, "ran out of its time budget") {
			return termsafe.CleanProseLine(n, 256)
		}
	}
	return "the check ran out of its time budget"
}

// incompleteLine is the person's line on a stop allowed because the check ran
// out of its time budget.
func incompleteLine(owed Owed) string {
	line := "abcd: the stop was allowed: " + incompleteDetail(owed)
	if item, ok := owed.Next(); ok {
		line += "; work read as owed so far: " + describeItem(item)
	}
	return line + "."
}

// presentLine is the person's line on a stop while present with work owed.
func presentLine(owed Owed) string {
	item, _ := owed.Next()
	return fmt.Sprintf("abcd: stopped with work owed (%s); next: %s.", count(len(owed.Items), "item", "items"), describeItem(item))
}

func clock(t time.Time, loc *time.Location) string {
	if loc == nil {
		loc = time.Local
	}
	return t.In(loc).Format("15:04 MST")
}
