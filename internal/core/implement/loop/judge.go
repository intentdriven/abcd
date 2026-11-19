package loop

// judge.go is the review set's judge (itd-2610240922461281,
// spc-2610240951109420 step 5): where a code-review role's two reviewers
// disagree, one passing the round's head and another failing it, the run's
// judge may strike from the fix brief a located finding it rules false, each
// with its reason, and do nothing else (decision 7, the product thinker,
// 2026-10-24: "Strike false findings only").
//
//   - The judge is launched only on a run whose stored judge is active
//     (reviewset.go: fixed when the run starts, waiting until a judge agent
//     clears itd-81's calibration gate), once every validator of the round has
//     returned, and only where a role's reviewers disagree: agreeing
//     reviewers, two failing verdicts included, start none. A waiting judge is
//     never launched, awaited or counted.
//   - A launch asks the calibration gate again when it launches and never rests
//     on the stored status alone: a judge whose agent no longer clears it is
//     not launched, the round records why (RoundJudge.Unlaunched), and the
//     round goes to its fix round with every finding.
//   - The judge is handed both reviewers' returns of each disagreeing role and
//     the round's head, at validate/round-<n>/judge/, and the located findings
//     of the role's failing reviewers, the only ones it may strike. Its return
//     opens with the head it read and lists each strike under one `### Struck`
//     heading, `- file:line — reason`; a strike of anything else, or with no
//     reason, is refused and the judge handed out again.
//   - A struck finding carries FindingStruck and the judge's reason, and is
//     left out of the fix brief, which names it as struck and not handed on.
//     The judge never passes a round: whatever its return says, the round goes
//     on to its fix round as the stricter verdict decides, and a round whose
//     every failing finding was struck goes to one whose brief hands on no
//     finding.

import (
	"bytes"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// RoleJudge is the role a review set's judge is handed out as.
const RoleJudge = "review-judge"

// JudgeDirName is the judge's directory in a round.
const JudgeDirName = "judge"

// FindingStruck is the status of a finding the round's judge struck.
const FindingStruck = "struck"

// maxStrikeReason bounds a strike's reason, in bytes.
const maxStrikeReason = 300

// maxUnlaunched bounds the stored reason a judge was not launched.
const maxUnlaunched = 500

// struckHeading is the heading a judge's return lists its strikes under.
var struckHeading = regexp.MustCompile(`(?m)^[ \t]{0,3}#{1,6}[ \t]+Struck[ \t]*\r?$`)

// RoundJudge is a round's judge: the roles whose reviewers disagreed, in the
// order the round hands them out; the class, model and effort the run's judge
// resolved to; its brief and return; whether the loop recorded its return
// (Ruled) and how many locations it struck; or, where the run's active judge
// was not launched because its agent no longer cleared the gate, why
// (Unlaunched), with no brief or return.
type RoundJudge struct {
	Roles      []string `json:"roles"`
	Class      string   `json:"class"`
	Model      string   `json:"model"`
	Effort     string   `json:"effort,omitempty"`
	Brief      string   `json:"brief,omitempty"`
	Return     string   `json:"return,omitempty"`
	Ruled      bool     `json:"ruled,omitempty"`
	Struck     int      `json:"struck,omitempty"`
	Unlaunched string   `json:"unlaunched,omitempty"`
}

// disagreeing are the code-review roles whose reviewers disagree in r, in the
// order the round hands them out.
func disagreeing(r ValidationRound) []string {
	var out []string
	for _, role := range reviewRoles {
		for _, c := range r.Comparisons {
			if c.Role == role && c.Outcome == ComparisonDisagree {
				out = append(out, role)
			}
		}
	}
	return out
}

// judgeMove is what a round whose every validator returned and which did not
// pass wants of its judge.
type judgeMove int

const (
	// judgeNone: no judge runs, or it has ruled, or it was not launched.
	judgeNone judgeMove = iota
	// judgeLaunch: the judge is due and not out.
	judgeLaunch
	// judgeOut: the judge is out.
	judgeOut
)

// judgeStep is what round r of lane l wants of its judge.
func judgeStep(st State, l Lane, r ValidationRound) judgeMove {
	if j := r.Judge; j != nil {
		switch {
		case j.Unlaunched != "" || j.Ruled:
			return judgeNone
		case l.awaited(j.Return):
			return judgeOut
		}
		return judgeLaunch
	}
	if st.Review == nil || !st.Review.Judge.Active() || len(disagreeing(r)) == 0 {
		return judgeNone
	}
	return judgeLaunch
}

// judgeCleared reports whether the judge's agent clears itd-81's calibration
// gate as the judge launches; nil gate is production's (Options.judgeGate).
func (c Context) judgeCleared() (bool, string) {
	if c.judgeGate != nil {
		return c.judgeGate()
	}
	return Options{}.judgeGate()
}

// launchJudge hands round r's judge out, recording it on r: its await and the
// record's note; or, where its agent no longer clears the gate, records why it
// was not launched and returns no await, so the fix round follows.
func launchJudge(c Context, lane Lane, r *ValidationRound) (*Await, string, error) {
	j := c.State.Review.Judge
	rj := &RoundJudge{Roles: disagreeing(*r), Class: j.Class, Model: j.Model, Effort: j.Effort}
	if r.Judge != nil {
		rj.Roles = slices.Clone(r.Judge.Roles)
	}
	if ok, why := c.judgeCleared(); !ok {
		rj.Unlaunched = plainBounded(why, maxUnlaunched)
		if rj.Unlaunched == "" {
			rj.Unlaunched = "its agent does not clear itd-81's calibration gate"
		}
		r.Judge = rj
		return nil, fmt.Sprintf("the run's judge, %s, was not launched on the disagreement over %s: %s; the stricter verdict decides",
			j.phrase(), strings.Join(rj.Roles, " and "), rj.Unlaunched), nil
	}
	dir, err := roundDir(c.State.RunID, lane.ID, r.Round, JudgeDirName)
	if err != nil {
		return nil, "", err
	}
	rj.Brief, rj.Return = dir+"/"+BriefFileName, dir+"/"+ReturnFileName
	r.Judge = rj
	if err := writeJudgeBrief(c, lane, *r); err != nil {
		return nil, "", err
	}
	a := Await{Role: RoleJudge, Brief: rj.Brief, Receipt: rj.Return, Class: rj.Class, Model: rj.Model, Effort: rj.Effort}
	return &a, fmt.Sprintf("round %d: the reviewers of %s disagree, so %s's head %s and their returns go to the run's judge, %s, which may strike a finding it rules false and passes nothing; awaiting its return at %s",
		r.Round, strings.Join(rj.Roles, " and "), lane.ID, shortSHA(r.HeadSHA), classPhrase(rj.Class, rj.Model, rj.Effort), rj.Return), nil
}

// plainBounded is s as one plain line of at most n bytes: "" where it holds a
// rune termsafe masks.
func plainBounded(s string, n int) string {
	s = strings.TrimSpace(s)
	if termsafe.Sanitize(s) != s || strings.ContainsAny(s, "\n\r\t") {
		return ""
	}
	if len(s) > n {
		s = strings.ToValidUTF8(s[:n], "")
	}
	return s
}

// strikable are the locations round r's judge may strike: each located
// finding of a failing reviewer of a role it was handed.
func strikable(r ValidationRound) map[location]bool {
	out := map[location]bool{}
	if r.Judge == nil {
		return out
	}
	for _, v := range r.Validators {
		if v.Pass || !slices.Contains(r.Judge.Roles, v.Role) {
			continue
		}
		for _, f := range v.Findings {
			out[f.at()] = true
		}
	}
	return out
}

// writeJudgeBrief renders the brief of round r's judge.
func writeJudgeBrief(c Context, lane Lane, r ValidationRound) error {
	j := r.Judge
	var b bytes.Buffer
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	p("# Judge brief: round %d of %s in %s\n\n", r.Round, lane.ID, c.State.RunID)
	p("You are the round's judge, a fresh agent at the %s class: you did not implement this lane and\n", classPhrase(j.Class, j.Model, j.Effort))
	p("did not review it. Two reviewers of one role disagree over the lane's head: one passed it and\n")
	p("another failed it. The stricter verdict decides, and it stands: you never pass a round, and\n")
	p("nothing you write passes one. What you may do is strike from the fix brief a located finding\n")
	p("you rule false, each with your reason; every finding you do not strike goes to the fix round.\n")
	p("No one will answer a question about this brief; what it does not settle, say so in your return.\n\n")
	p("- the lane: %s, its worktree `%s`, branch `%s`\n", lane.ID, lane.Worktree, lane.Branch)
	p("- the round's head: `%s`; the lane's diff: `%s..%s` (`git -C %s %s %s..%s`)\n", r.HeadSHA, lane.BaseSHA, r.HeadSHA, lane.Worktree, readerDiff, lane.BaseSHA, r.HeadSHA)
	for _, role := range j.Roles {
		p("- the returns of %s:\n", role)
		for _, v := range r.Validators {
			if v.Role != role {
				continue
			}
			who := v.Role
			if ph := v.phrase(); ph != "" {
				who += " (" + ph + ")"
			}
			p("  - %s, %s: `%s`\n", who, v.Verdict, abs(c.RepoRoot, v.Return))
			if !v.Pass && len(v.Findings) > 0 {
				p("    - the located findings you may strike: %s\n", strings.Join(findingLocations(v.Findings), ", "))
			}
		}
	}
	p("\nRead only; change nothing in the worktree or on the branch.\n\n")
	p("## What you hand back\n\n")
	p("Its first line reads exactly `Reviewed head: %s`, nothing before it.\n\n", r.HeadSHA)
	p("List each finding you strike under one `### Struck` heading, a bullet of its own: its `file:line`\n")
	p("first, exactly as listed above, then on the same line your reason, at most %d bytes of plain\n", maxStrikeReason)
	p("text (`- a.go:3 — why it is false`). Strike only a finding listed above: a return striking any\n")
	p("other, or a strike with no reason, is refused and handed out again. To strike none, write no\n")
	p("`### Struck` section, or leave it empty. Write no verdict: a judge states none, and none you\n")
	p("write is read.\n\n")
	p("Write your whole return to:\n\n`%s`\n", abs(c.RepoRoot, j.Return))
	return writeRoundFile(c.RepoRoot, j.Brief, b.Bytes())
}

// strike is one finding a judge's return strikes, with its reason.
type strike struct {
	at     location
	reason string
}

// strikeReason is what follows a strike's location on its line, less what
// stands between the two: the marks closing the location, then a dash, a
// colon or nothing. It reads one string and no fence.
func strikeReason(rest string) string {
	rest = strings.TrimLeft(rest, "*_`'\")]>")
	rest = strings.TrimLeft(rest, " \t")
	rest = strings.TrimLeft(rest, "\u2014\u2013:-")
	return strings.TrimSpace(rest)
}

// readStrikes reads the strikes a judge's return lists under its Struck
// headings: each top-level bullet leading with a `file:line` and its reason
// on the same line. A bullet leading with no location, or with no reason or
// one past maxStrikeReason or carrying a rune termsafe masks, is refused. A
// location struck twice keeps its first reason.
func readStrikes(text, worktree string) ([]strike, error) {
	var out []strike
	var fault error
	seen := map[location]bool{}
	eachSectionLine(relativeTo(text, worktree), struckHeading, func(ln string) {
		m := leadRe.FindStringIndex(ln)
		if fault != nil || m == nil {
			return
		}
		rest := ln[m[1]:]
		c := leadCitationRe.FindStringSubmatchIndex(rest)
		if c == nil {
			fault = fmt.Errorf("a struck bullet leads with no file:line: %q", termsafe.Sanitize(firstBytes(ln, 80)))
			return
		}
		l, ok := toLocation(rest[c[2]:c[3]], rest[c[4]:c[5]])
		if !ok {
			fault = fmt.Errorf("a struck bullet leads with no plain repository file:line: %q", termsafe.Sanitize(firstBytes(ln, 80)))
			return
		}
		reason := strikeReason(rest[c[1]:])
		switch {
		case reason == "":
			fault = fmt.Errorf("the strike of %s:%d gives no reason", l.file, l.line)
		case len(reason) > maxStrikeReason:
			fault = fmt.Errorf("the strike of %s:%d gives a reason past %d bytes", l.file, l.line, maxStrikeReason)
		case plainBounded(reason, maxStrikeReason) != reason:
			fault = fmt.Errorf("the strike of %s:%d gives a reason carrying a control, bidi or zero-width character", l.file, l.line)
		}
		if fault != nil || seen[l] {
			return
		}
		seen[l] = true
		out = append(out, strike{l, reason})
	})
	return out, fault
}

// firstBytes is at most n bytes of s, cut at a rune.
func firstBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.ToValidUTF8(s[:n], "")
}

// verifyJudge records round cur's judge's return: each finding it struck
// carries FindingStruck and the judge's reason, and the judge has ruled. A
// return that does not state the head, or strikes a finding it was not
// handed, or gives a strike no reason, is refused and nothing recorded. The
// round's verdicts are left as they are: the judge passes nothing.
func verifyJudge(c Context, lane *Lane, cur *ValidationRound, receiptRel string) error {
	j := cur.Judge
	if j == nil || j.Unlaunched != "" || j.Ruled || j.Return != receiptRel {
		return refuse("receipt", "", lane.ID, fmt.Sprintf("no judge of round %d awaits %s", cur.Round, receiptRel),
			"hand back the return `abcd implement step` names")
	}
	raw, err := readReturn(c.RepoRoot, receiptRel, lane.ID)
	if err != nil {
		return err
	}
	text := landing.NormalizeLineBreaks(string(raw))
	again := "start a fresh " + RoleJudge + " with the brief " + j.Brief + "; its first line reads exactly `Reviewed head: " + cur.HeadSHA +
		"`, and each strike under its `### Struck` heading is `- file:line — reason` for a finding the brief lists, then hand it back"
	if err := landing.StatesFullPair(text, cur.HeadSHA); err != nil {
		return refuse("receipt", "", lane.ID, fmt.Sprintf("%s does not state the head it read: %v", receiptRel, err), again)
	}
	strikes, err := readStrikes(text, lane.Worktree)
	if err != nil {
		return refuse("receipt", "", lane.ID, fmt.Sprintf("%s strikes no finding the loop can record: %v", receiptRel, err), again)
	}
	may := strikable(*cur)
	reasons := make(map[location]string, len(strikes))
	for _, s := range strikes {
		if !may[s.at] {
			return refuse("receipt", "", lane.ID, fmt.Sprintf("%s strikes %s:%d, which no failing reviewer of the %s cites: a judge strikes only a finding it was handed",
				receiptRel, s.at.file, s.at.line, strings.Join(j.Roles, " or ")), again)
		}
		reasons[s.at] = s.reason
	}
	cur.Validators = slices.Clone(cur.Validators)
	for i := range cur.Validators {
		v := &cur.Validators[i]
		if v.Pass || !slices.Contains(j.Roles, v.Role) || len(v.Findings) == 0 {
			continue
		}
		v.Findings = slices.Clone(v.Findings)
		for k := range v.Findings {
			if r, ok := reasons[v.Findings[k].at()]; ok {
				v.Findings[k].Status, v.Findings[k].Reason = FindingStruck, r
			}
		}
	}
	ruled := *j
	ruled.Ruled, ruled.Struck = true, len(reasons)
	cur.Judge = &ruled
	return nil
}

// judgeNote is the run record's words for round r's judge's return.
func judgeNote(r ValidationRound) string {
	j := r.Judge
	var struck []string
	seen := map[location]bool{}
	for _, v := range r.Validators {
		for _, f := range v.Findings {
			if f.Status == FindingStruck && !seen[f.at()] {
				seen[f.at()] = true
				struck = append(struck, fmt.Sprintf("%s:%d", f.File, f.Line))
			}
		}
	}
	what := "struck no finding"
	if len(struck) > 0 {
		what = fmt.Sprintf("struck %d located finding(s) as false (%s), each with its reason on the finding", j.Struck, strings.Join(struck, ", "))
	}
	return fmt.Sprintf("the round's judge, %s, ruled on the disagreement over %s and %s; it passes nothing, so the stricter verdict decides and the round goes to a fix round",
		classPhrase(j.Class, j.Model, j.Effort), strings.Join(j.Roles, " and "), what)
}

// struckFindings and handedFindings split a reviewer's located findings by
// whether the round's judge struck them.
func struckFindings(fs []Finding) []Finding {
	return slices.DeleteFunc(slices.Clone(fs), func(f Finding) bool { return f.Status != FindingStruck })
}

func handedFindings(fs []Finding) []Finding {
	return slices.DeleteFunc(slices.Clone(fs), func(f Finding) bool { return f.Status == FindingStruck })
}

// handsOnNothing reports whether the judge struck every finding the failing
// validators of round r would hand to its fix round: each is a reviewer of a
// role the judge was handed, citing at least one location, every one kept
// and struck.
func handsOnNothing(r ValidationRound, failing []ValidatorRun) bool {
	if r.Judge == nil || !r.Judge.Ruled || r.Judge.Struck == 0 || len(failing) == 0 {
		return false
	}
	for _, v := range failing {
		if !slices.Contains(r.Judge.Roles, v.Role) || v.Located == nil || *v.Located == 0 || len(v.Findings) != *v.Located ||
			len(handedFindings(v.Findings)) > 0 {
			return false
		}
	}
	return true
}

// judged reports the first thing, in a state, that only version 28 and later
// write: a round's judge or a struck finding; "" when there is none.
func (s State) judged() string {
	for _, l := range s.Lanes {
		for _, r := range l.Validation {
			if r.Judge != nil {
				return fmt.Sprintf("a judge on %s's round %d", l.ID, r.Round)
			}
			for i, v := range r.Validators {
				for _, f := range v.Findings {
					if f.Status == FindingStruck || f.Reason != "" {
						return fmt.Sprintf("a judge's strike on %s's round %d validator %d", l.ID, r.Round, i+1)
					}
				}
			}
		}
	}
	return ""
}

// judgesWritten reports whether every round's judge and every strike a state
// carries is one the loop writes: a judge only on a run whose judge was
// active, at its class, model and effort, over roles whose reviewers disagree
// in the round, and either not launched with its reason, or at the round's
// judge paths; a strike only on a finding of a failing reviewer of a role the
// judge was handed, once it ruled, with a plain reason, and a reason on no
// other finding; and the count the judge struck, where the round keeps its
// findings, the locations struck.
func (s State) judgesWritten() bool {
	for _, l := range s.Lanes {
		for _, r := range l.Validation {
			j := r.Judge
			struck := map[location]bool{}
			for _, v := range r.Validators {
				for _, f := range v.Findings {
					if f.Status != FindingStruck {
						if f.Reason != "" {
							return false
						}
						continue
					}
					if j == nil || !j.Ruled || v.Pass || !slices.Contains(j.Roles, v.Role) || f.Reason == "" ||
						plainBounded(f.Reason, maxStrikeReason) != f.Reason {
						return false
					}
					struck[f.at()] = true
				}
			}
			if j == nil {
				continue
			}
			rj := (*ReviewJudge)(nil)
			if s.Review != nil {
				rj = s.Review.Judge
			}
			if !rj.Active() || j.Class != rj.Class || j.Model != rj.Model || j.Effort != rj.Effort {
				return false
			}
			want := disagreeing(r)
			if len(j.Roles) == 0 || !slices.Equal(j.Roles, want) {
				return false
			}
			if j.Unlaunched != "" {
				if plainBounded(j.Unlaunched, maxUnlaunched) != j.Unlaunched || j.Brief != "" || j.Return != "" || j.Ruled || j.Struck != 0 {
					return false
				}
				continue
			}
			dir, err := roundDir(s.RunID, l.ID, r.Round, JudgeDirName)
			if err != nil || j.Brief != dir+"/"+BriefFileName || j.Return != dir+"/"+ReturnFileName {
				return false
			}
			if !j.Ruled && j.Struck != 0 || j.Struck < 0 || j.Struck > maxLocatedFindings*maxReviewers*len(reviewRoles) {
				return false
			}
			if r.carriesFindings() && j.Struck != len(struck) {
				return false
			}
		}
	}
	return true
}

// judgeAwaitWritten reports whether a's class, model and effort are the ones
// the lane's round records for the judge whose return a awaits.
func judgeAwaitWritten(l Lane, a Await) bool {
	for _, r := range l.Validation {
		if j := r.Judge; j != nil && j.Unlaunched == "" && j.Return == a.Receipt {
			return a.Reviewer == nil && a.Brief == j.Brief && a.Class == j.Class && a.Model == j.Model && a.Effort == j.Effort
		}
	}
	return false
}
