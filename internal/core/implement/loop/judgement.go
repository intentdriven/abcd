package loop

// judgement.go is the host judgement before a drain opens a lane (itd-82
// scope 3 and criterion 2; spc-2609212015054359 step 1). An issue the field
// rule lets through still has one question asked of it before its lane opens:
// does its remedy, carried out as written, change what a user sees or a trust
// boundary? The question is the host's (adr-25), asked the host-pass way: the
// drain writes a request file in the local tier naming the issue, its remedy
// and the answer's shape, records that it awaits the answer, and opens
// nothing; the host writes its answer where the request says and hands it back
// with `abcd drain --judgement <file>`, which validates it strictly before it
// changes anything.
//
// The judgement may only hand an issue back, never let one through. A yes
// routes the issue exactly as a lane's hand-back of the same kind is routed (a
// user-visible change promoted to an intent draft, a trust rule flagged as
// needing a decision record with the reason as its question), and no lane
// opens for it; a no changes nothing, and the lane opens as the fields already
// allow. An answer over an issue that has left the eligible set since the
// request is recorded and applied to nothing, and an answer over a remedy that
// has been rewritten since is refused, the next move asking again. Every
// answer is recorded in the drain's state with the disposition it decided,
// and nothing is written onto the issue for it (decision 8).

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// DrainJudgementRequestRel is the request the drain writes for the host, and
// DrainJudgementAnswerRel where the host writes its answer. One judgement is
// awaited at a time, so each is one file beside the drain's state.
const (
	DrainJudgementRequestRel = RunRelDir + "/drain-judgement.request.md"
	DrainJudgementAnswerRel  = RunRelDir + "/drain-judgement.json"
)

// DrainJudgementSchemaVersion is the answer's shape.
const DrainJudgementSchemaVersion = 1

// maxJudgementBytes caps the answer read.
const maxJudgementBytes = 64 << 10

// The answers to the one question.
const (
	// JudgementYes: the remedy changes what a user sees or a trust boundary;
	// the issue is handed back.
	JudgementYes = "yes"
	// JudgementNo: it changes neither; the lane opens.
	JudgementNo = "no"
)

// DrainFromJudgement marks a hand-back the host judgement made, in a
// DrainRoute's From.
const DrainFromJudgement = "judgement"

// DrainFromReservation is a hand-back's From when the drain could not reserve
// the issue: its found_at names no path to reserve it with.
const DrainFromReservation = "reservation"

// judgementKinds are the kinds a yes names, each routed as a lane's hand-back
// of that kind is.
var judgementKinds = []string{HandBackUserVisible, HandBackTrustRule}

// DrainJudging is the judgement the drain awaits before it opens an issue's
// lane: the issue, the digest of the remedy the request showed, and where the
// request and the answer are.
type DrainJudging struct {
	Issue        string    `json:"issue"`
	RemedySHA256 string    `json:"remedy_sha256"`
	Request      string    `json:"request"`
	Answer       string    `json:"answer"`
	RequestedAt  time.Time `json:"requested_at"`
	// Hold is the reservation the drain took on the issue to ask this
	// judgement; nil when it took none (no origin, or this person already
	// held the issue from this copy, which is theirs and never the drain's
	// to release).
	Hold *DrainHold `json:"hold,omitempty"`
	// RoutedAsked is true once the routed judge was started on this request
	// and did not answer (drain_judge.go): the polls that follow while the
	// host's request stands do not start it again. A judge that never
	// started (its directory not made, its request refused before any
	// launch) leaves it false, so the next move starts it.
	RoutedAsked bool `json:"routed_asked,omitempty"`
}

// DrainHold is a reservation the drain took on an issue to judge it: the
// commit its push made, and when it was claimed and lapses. The drain releases
// that commit and no other, so a reservation this person made or renewed by
// hand since is left as it stands. ForkUnchecked and ForkAllowed are the fork
// check the act could not make or waived (claim's Result), kept so the lane a
// no opens on the hold says so as a lane that reserved for itself does
// (spc-2610161044461448 step 10).
type DrainHold struct {
	Commit        string    `json:"commit"`
	ClaimedAt     time.Time `json:"claimed_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	ForkUnchecked string    `json:"fork_unchecked,omitempty"`
	ForkAllowed   string    `json:"fork_allowed,omitempty"`
}

// DrainJudgement is one answer the drain took: the question's answer, the kind
// a yes names, the host's reason, and whether it decided the issue's
// disposition (Applied) or, the issue having left the eligible set since the
// request, decided nothing (Note says why).
type DrainJudgement struct {
	Issue        string    `json:"issue"`
	RemedySHA256 string    `json:"remedy_sha256"`
	Answer       string    `json:"answer"`
	Kind         string    `json:"kind,omitempty"`
	Reason       string    `json:"reason"`
	Applied      bool      `json:"applied"`
	Note         string    `json:"note,omitempty"`
	At           time.Time `json:"at"`
	// Hold is the reservation the answer was asked on, nil when the drain
	// took none: the lane a no opens on it, in this move or a later one,
	// names the fork check it could not make or waived.
	Hold *DrainHold `json:"hold,omitempty"`
	// Route is the runner that answered, when the drain's judge is routed to
	// one (drain_judge.go); nil for the host's answer.
	Route *runner.RouteRecord `json:"route,omitempty"`
}

// JudgementAnswer is the host's answer file, read strictly: any other field
// refuses it.
type JudgementAnswer struct {
	SchemaVersion int    `json:"schema_version"`
	Issue         string `json:"issue"`
	RemedySHA256  string `json:"remedy_sha256"`
	Answer        string `json:"answer"`
	Kind          string `json:"kind,omitempty"`
	Reason        string `json:"reason"`
}

// remedyDigest is the hex sha256 of a remedy as the ledger reader gives it.
func remedyDigest(remedy string) string {
	sum := sha256.Sum256([]byte(remedy))
	return hex.EncodeToString(sum[:])
}

// judgementFor is the latest answer the drain took on issue over the remedy
// whose digest is given, if any. An answer taken no later than the drain last handed
// the issue back judged a record since changed, which is why the drain is
// considering it again (drain_readmit.go), so it decides nothing.
func judgementFor(st DrainState, issue, digest string) *DrainJudgement {
	since := st.handedBackAt(issue)
	for i := len(st.Judgements) - 1; i >= 0; i-- {
		j := st.Judgements[i]
		if j.Issue == issue && j.RemedySHA256 == digest && j.Applied {
			if since != nil && !j.At.After(*since) {
				return nil
			}
			return &j
		}
	}
	return nil
}

// requestJudgement writes the request for iss's judgement and returns what
// the drain then awaits. When prior already awaits this issue over this
// remedy, the request is written again and the answer the host may have
// written is kept; otherwise any answer an earlier request left is removed,
// so it can never be taken for this one.
func requestJudgement(repoRoot string, iss capture.Issue, v capture.DrainVerdict, prior *DrainJudging, now time.Time) (DrainJudging, error) {
	j := DrainJudging{Issue: iss.ID, RemedySHA256: remedyDigest(iss.Remedy), Request: DrainJudgementRequestRel,
		Answer: DrainJudgementAnswerRel, RequestedAt: now}
	same := prior != nil && prior.Issue == j.Issue && prior.RemedySHA256 == j.RemedySHA256
	if same {
		j.RequestedAt, j.RoutedAsked = prior.RequestedAt, prior.RoutedAsked
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return DrainJudging{}, fmt.Errorf("opening the checkout: %w", err)
	}
	defer root.Close()
	if !same {
		if err := root.Remove(DrainJudgementAnswerRel); err != nil && !errors.Is(err, os.ErrNotExist) {
			return DrainJudging{}, fmt.Errorf("removing the earlier judgement answer %s: %w", DrainJudgementAnswerRel, err)
		}
	}
	if err := fsutil.WriteFileAtomicInRoot(root, DrainJudgementRequestRel, []byte(judgementRequest(iss, v, j, false)), filePerm); err != nil {
		return DrainJudging{}, fmt.Errorf("writing the judgement request %s: %w", DrainJudgementRequestRel, err)
	}
	return j, nil
}

// judgementMove is what the caller is told while a judgement is awaited.
func judgementMove(j DrainJudging) string {
	return fmt.Sprintf("judge %s's remedy before its lane opens: read %s, write the answer it asks for to %s, then run `abcd drain --judgement %s`; until then the drain opens nothing",
		j.Issue, j.Request, j.Answer, j.Answer)
}

// judgementQuoteNote tells the request's reader the substitutions the
// request makes in the record text it quotes: the termsafe mask, then
// fenceQuote's.
const judgementQuoteNote = "The remedy and the record are quoted as the ledger holds them, save two substitutions: a\n" +
	"control character or a rune that renders as nothing (a bidi override, a zero-width or tag rune) is\n" +
	"written `?`, and an HTML comment opener or closer inside a quote has its second hyphen written\n" +
	"`&#45;`, so no quote can end its fence early.\n\n"

// judgementRequest is the request's text: the question, what each answer
// does, the issue with its remedy and record, and the answer's shape.
//
// The request is an agent's input, so every record string it quotes is
// masked (termsafe.Sanitize for a one-line field and the remedy,
// termsafe.SanitizeBlock for the body): a bidi override or a tag rune in a
// record would otherwise reach the judging agent unseen by the person
// (iss-2610121653499543). The digest is not: j.RemedySHA256 is over the
// remedy as stored, so a remedy rewritten with a hidden rune where a '?'
// stood still changes it.
//
// routed is the brief the drain's routed judge follows (drain_judge.go): its
// answer goes where j.Answer says, carries no digest, which abcd computes
// itself, and is taken by the move that asked it, so it names no verb to run.
func judgementRequest(iss capture.Issue, v capture.DrainVerdict, j DrainJudging, routed bool) string {
	path := termsafe.Sanitize(iss.Path)
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	p("# Drain judgement request: %s\n\n", iss.ID)
	p("`abcd drain` is about to open a lane that fixes %s alone, unattended. Its fields say it needs no\n", iss.ID)
	p("decision: %s. One question is left, and it is yours. Judge from this request alone: do not\n", v.Reason)
	p("open the lane, edit the record or fix anything.\n\n")
	p("**The question:** does this remedy, carried out as written, change what a user sees, or a trust\n")
	p("boundary?\n\n")
	p("- **What a user sees**: anything a person reads or relies on, such as a command's output or exit\n")
	p("  code, a flag, a message, a page, a prompt, or what a command does when a person runs it.\n")
	p("- **A trust boundary**: what is allowed or refused, read, written, run or sent, whose input is\n")
	p("  trusted, or a rule about safety, secrets or permissions.\n\n")
	p("Answer `yes` when it changes either, and name the kind: `%s` for what a user sees, `%s` for a\n", HandBackUserVisible, HandBackTrustRule)
	p("trust boundary, and `%s` when it changes both. When the remedy does not let you tell, answer\n", HandBackTrustRule)
	p("`yes` and say what you could not tell. Answer `no` only when it changes neither.\n\n")
	p("A `no` changes nothing: the lane opens as the issue's fields already allow. A `yes` hands the issue\n")
	p("back to a person before any lane opens: `%s` promotes it to an intent draft for a person to plan,\n", HandBackUserVisible)
	p("and `%s` flags it as needing a decision record, with your reason as the question that record\n", HandBackTrustRule)
	p("must answer. The judgement can only ever hand an issue back.\n\n")
	p("## The issue\n\n")
	p("- id: %s\n", iss.ID)
	p("- record: %s\n", path)
	p("- severity: %s\n", termsafe.Sanitize(string(iss.Severity)))
	p("- category: %s\n\n", termsafe.Sanitize(string(iss.Category)))
	b.WriteString(judgementQuoteNote)
	p("### The remedy\n\n<!-- begin remedy -->\n\n%s\n\n<!-- end remedy -->\n\n", fenceQuote(termsafe.Sanitize(iss.Remedy)))
	p("### The record's body\n\n<!-- begin %s -->\n\n%s\n\n<!-- end %s -->\n\n", path, fenceQuote(termsafe.SanitizeBlock(strings.TrimSpace(iss.Body))), path)
	p("## Your answer\n\n")
	p("Write this JSON to `%s`, strictly in this shape (any other field refuses it):\n\n", j.Answer)
	p("    {\n      \"schema_version\": %d,\n      \"issue\": %q,\n", DrainJudgementSchemaVersion, iss.ID)
	if !routed {
		p("      \"remedy_sha256\": %q,\n", j.RemedySHA256)
	}
	p("      \"answer\": \"yes or no\",\n      \"kind\": \"%s or %s, with yes only; leave the field out with no\",\n", HandBackUserVisible, HandBackTrustRule)
	p("      \"reason\": \"one sentence; with %s, the question the decision record must answer\"\n    }\n\n", HandBackTrustRule)
	if routed {
		p("Write nothing else and run nothing: abcd reads the answer once you have written it.\n")
		return b.String()
	}
	p("`remedy_sha256` is the digest of the remedy as the ledger holds it; copy it as it stands. Then run\n")
	p("`abcd drain --judgement %s`. A remedy rewritten in the meantime refuses the answer, and the next\n", j.Answer)
	p("`abcd drain` asks again.\n")
	return b.String()
}

// judgementGaps names what an answer is missing against the judgement it
// answers. The values are the host's, so a refused one is described, never
// quoted.
func judgementGaps(a JudgementAnswer, j DrainJudging) []string {
	var gaps []string
	if a.SchemaVersion != DrainJudgementSchemaVersion {
		gaps = append(gaps, fmt.Sprintf("schema_version %d", DrainJudgementSchemaVersion))
	}
	if a.Issue != j.Issue {
		gaps = append(gaps, "issue "+j.Issue+", the issue the request names (it names "+termsafe.DescribeRefused(a.Issue)+")")
	}
	if a.RemedySHA256 != j.RemedySHA256 {
		gaps = append(gaps, "remedy_sha256 as the request states it, the digest of the remedy as the ledger held it")
	}
	switch a.Answer {
	case JudgementYes:
		if !validJudgementKind(a.Kind) {
			gaps = append(gaps, "kind with yes, one of "+strings.Join(judgementKinds, ", ")+" (it names "+termsafe.DescribeRefused(a.Kind)+")")
		}
	case JudgementNo:
		if a.Kind != "" {
			gaps = append(gaps, "no kind with no: a no hands nothing back")
		}
	default:
		gaps = append(gaps, "answer, yes or no (it names "+termsafe.DescribeRefused(a.Answer)+")")
	}
	if strings.TrimSpace(a.Reason) == "" || len(a.Reason) > maxResolutionText {
		gaps = append(gaps, fmt.Sprintf("reason, present and within %d bytes", maxResolutionText))
	}
	return gaps
}

func validJudgementKind(kind string) bool {
	for _, k := range judgementKinds {
		if kind == k {
			return true
		}
	}
	return false
}

// takeJudgement reads and validates the host's answer at path against the
// judgement st awaits, under the plan as this move read it. It refuses, with
// nothing written, when no judgement is awaited, when path is not the answer
// the drain named, when the answer is not the strict shape or does not answer
// the request, and when the remedy has been rewritten since the request. An
// issue that has left the eligible set since is answered with a judgement
// applied to nothing.
func takeJudgement(repoRoot, path string, st DrainState, live bool, plan capture.DrainPlan, now time.Time) (DrainJudgement, error) {
	if !live || st.Judging == nil {
		return DrainJudgement{}, refuse(StageDrain, "", "", "the drain awaits no judgement, so there is nothing for --judgement to answer",
			"run `abcd drain`; it writes a judgement request when an eligible issue's lane is next")
	}
	j := *st.Judging
	if !samePath(repoRoot, path, j.Answer) {
		return DrainJudgement{}, refuse(StageDrain, "", "", "the drain awaits the judgement on "+j.Issue+" at "+j.Answer+", not at the path given",
			"write the answer there and run `abcd drain --judgement "+j.Answer+"`")
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return DrainJudgement{}, fmt.Errorf("opening the checkout: %w", err)
	}
	defer root.Close()
	data, err := fsutil.ReadGuardedInRoot(root, j.Answer, maxJudgementBytes)
	if err != nil {
		return DrainJudgement{}, refuse(StageDrain, "", "", fmt.Sprintf("%s cannot be read as the judgement's answer: %v", j.Answer, err),
			"write the answer as a regular file of at most "+fmt.Sprint(maxJudgementBytes)+" bytes, then run `abcd drain --judgement "+j.Answer+"`")
	}
	var a JudgementAnswer
	if err := jsonstrict.Decode(data, &a); err != nil {
		return DrainJudgement{}, refuse(StageDrain, "", "", j.Answer+" does not parse as the judgement's answer: "+termsafe.Sanitize(err.Error()),
			"write the shape "+j.Request+" states, no other field, then run `abcd drain --judgement "+j.Answer+"`")
	}
	if gaps := judgementGaps(a, j); len(gaps) > 0 {
		return DrainJudgement{}, refuse(StageDrain, "", "", j.Answer+" is missing "+strings.Join(gaps, "; "),
			"correct the answer as "+j.Request+" states it, then run `abcd drain --judgement "+j.Answer+"`")
	}
	return judgeAnswer(a, j, plan, now)
}

// judgeAnswer is the judgement a valid answer a to j makes under plan: one
// applied to nothing when the issue has left the eligible set since the
// request, a refusal when its remedy has been rewritten since, and otherwise
// the issue's disposition.
func judgeAnswer(a JudgementAnswer, j DrainJudging, plan capture.DrainPlan, now time.Time) (DrainJudgement, error) {
	out := DrainJudgement{Issue: j.Issue, RemedySHA256: j.RemedySHA256, Answer: a.Answer, Kind: a.Kind,
		Reason: termsafe.Sanitize(strings.TrimSpace(a.Reason)), At: now}
	var verdict *capture.DrainVerdict
	for i := range plan.Dispositions {
		if plan.Dispositions[i].ID == j.Issue {
			verdict = &plan.Dispositions[i]
		}
	}
	if verdict == nil || verdict.Outcome != capture.DrainEligible {
		// The judgement never makes an issue eligible: it is recorded, and
		// decides nothing.
		why := "it is no longer open"
		if verdict != nil {
			why = fmt.Sprintf("%s (%s): %s", verdict.Outcome, verdict.Rule, verdict.Reason)
		}
		out.Note = j.Issue + " is no longer eligible, so the judgement decides nothing: " + why
		return out, nil
	}
	if iss, ok := plan.OpenIssue(j.Issue); !ok || remedyDigest(iss.Remedy) != j.RemedySHA256 {
		return DrainJudgement{}, refuse(StageDrain, "", "", j.Issue+"'s remedy has been rewritten since the judgement was requested, so the answer judges a remedy the lane would not work from",
			"run `abcd drain`; it asks again over the remedy as it stands")
	}
	out.Applied = true
	return out, nil
}

// JudgementSummary is a judgement in one line, for the text surfaces.
func JudgementSummary(j DrainJudgement) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s", j.Issue, j.Answer)
	if j.Kind != "" {
		fmt.Fprintf(&b, " (%s)", j.Kind)
	}
	fmt.Fprintf(&b, ": %s", j.Reason)
	switch {
	case !j.Applied:
		fmt.Fprintf(&b, "; %s", j.Note)
	case j.Answer == JudgementYes:
		b.WriteString("; handed back before any lane opened")
	default:
		b.WriteString("; its lane may open")
	}
	return b.String()
}
