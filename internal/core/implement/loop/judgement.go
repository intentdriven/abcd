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
// whose digest is given, if any.
func judgementFor(st DrainState, issue, digest string) *DrainJudgement {
	for i := len(st.Judgements) - 1; i >= 0; i-- {
		j := st.Judgements[i]
		if j.Issue == issue && j.RemedySHA256 == digest && j.Applied {
			return &j
		}
	}
	return nil
}

// openIssues reads the open ledger by id.
func openIssues(repoRoot string) (map[string]capture.Issue, error) {
	lr, err := capture.List(capture.ListRequest{RepoRoot: repoRoot, State: capture.StateOpen})
	if err != nil {
		return nil, err
	}
	out := make(map[string]capture.Issue, len(lr.Issues))
	for _, iss := range lr.Issues {
		out[iss.ID] = iss
	}
	return out, nil
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
		j.RequestedAt = prior.RequestedAt
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
	if err := fsutil.WriteFileAtomicInRoot(root, DrainJudgementRequestRel, []byte(judgementRequest(iss, v, j)), filePerm); err != nil {
		return DrainJudging{}, fmt.Errorf("writing the judgement request %s: %w", DrainJudgementRequestRel, err)
	}
	return j, nil
}

// judgementMove is what the caller is told while a judgement is awaited.
func judgementMove(j DrainJudging) string {
	return fmt.Sprintf("judge %s's remedy before its lane opens: read %s, write the answer it asks for to %s, then run `abcd drain --judgement %s`; until then the drain opens nothing",
		j.Issue, j.Request, j.Answer, j.Answer)
}

// fence is a code fence longer than any backtick run in s, so quoted record
// text cannot close it early.
func fence(s string) string {
	longest, run := 0, 0
	for _, r := range s {
		if r == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	return strings.Repeat("`", max(3, longest+1))
}

// judgementRequest is the request's text: the question, what each answer
// does, the issue with its remedy and record, and the answer's shape.
func judgementRequest(iss capture.Issue, v capture.DrainVerdict, j DrainJudging) string {
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
	p("- record: %s\n", iss.Path)
	p("- severity: %s\n", iss.Severity)
	p("- category: %s\n\n", iss.Category)
	f := fence(iss.Remedy)
	p("### The remedy\n\n%stext\n%s\n%s\n\n", f, iss.Remedy, f)
	body := strings.TrimSpace(iss.Body)
	f = fence(body)
	p("### The record's body\n\n%stext\n%s\n%s\n\n", f, body, f)
	p("## Your answer\n\n")
	p("Write this JSON to `%s`, strictly in this shape (any other field refuses it):\n\n", j.Answer)
	p("```json\n{\n  \"schema_version\": %d,\n  \"issue\": %q,\n  \"remedy_sha256\": %q,\n", DrainJudgementSchemaVersion, iss.ID, j.RemedySHA256)
	p("  \"answer\": \"yes or no\",\n  \"kind\": \"%s or %s, with yes only; leave the field out with no\",\n", HandBackUserVisible, HandBackTrustRule)
	p("  \"reason\": \"one sentence; with %s, the question the decision record must answer\"\n}\n```\n\n", HandBackTrustRule)
	p("`remedy_sha256` is the remedy's digest as this request showed it; copy it as it stands. Then run\n")
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
		gaps = append(gaps, "remedy_sha256 as the request states it, the digest of the remedy it showed")
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
	issues, err := openIssues(repoRoot)
	if err != nil {
		return DrainJudgement{}, err
	}
	if iss, ok := issues[j.Issue]; !ok || remedyDigest(iss.Remedy) != j.RemedySHA256 {
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
