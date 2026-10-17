package loop

// drain_judge.go is the drain's judge routed to a runner (spc-2610161044461448
// step 14; iss-2610161015422306; dcn-2610161040390001). The judgement over an
// issue's remedy is a role, drain-judge, on the same routing the build roles
// take (roles.drain-judge.runner, the person's own layer alone): routed to a
// runner, the move that writes the host's request also hands the runner a
// brief of its own, takes its answer under the same validation a host's
// answer passes, and goes on as `--judgement` would have, so no person writes
// the answer. The routed answer carries no digest: abcd computes the digest
// of the remedy it showed, and the move that asked is the move that takes it,
// so the remedy cannot have been rewritten in between.
//
// A runner that is absent, refuses, fails, answers unparsably or writes an
// answer the validation refuses is a fallback, recorded in the drain's state
// (JudgeFallbacks, the latest maxJudgeFallbacks kept), and the host's
// request, written beside it, stands, as it does with no route; the runner is
// started once per request (DrainJudging.RoutedAsked), so a poll while the
// host's request stands starts no runner and records no receipt; a rate limit or any other failure leaves the request
// standing too, with the reason in the move's next step. The judge never
// stops the drain.
//
// The runner is started under the drain's lock, so another `abcd drain` in
// this checkout backs off while it judges; the run is bounded by
// drainJudgeTimeout, a small part of the reservation's lease the drain took
// to ask it, and by the drain's own context (DrainOptions.Context): a drain
// interrupted or terminated while its judge runs kills the runner's process
// group before the move returns, records no fallback for the stop, and
// refuses the move saying so, so the judge is asked again by the next move.
//
// The judge runs in a directory of its own (judgeDir): a new private
// directory outside every working tree, holding the brief and, once it
// answers, its answer, and removed when the move has read it. The judge may
// read only that directory and write only its answer file there, and is
// refused reading or writing anywhere in the checkout (drainJudgeGrants); the
// checkout is otherwise named to the runner only as the place whose programs
// are never run: the answer steers an unattended lane, and the brief quotes a
// record a stranger may have written, so the judge decides from the brief
// alone and can change nothing the drain or a peer relies on.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// RoleDrainJudge is the drain's judge, the role roles.drain-judge.runner
// routes.
const RoleDrainJudge = "drain-judge"

// drainJudgeBriefName is the brief the routed judge follows, and
// drainJudgeAnswerName where it writes its answer, each in the judge's own
// directory (judgeDir): apart from the host's, so an answer a runner left can
// never be taken for the host's.
const (
	drainJudgeBriefName  = "drain-judgement.md"
	drainJudgeAnswerName = "drain-judgement.json"
)

// maxJudgeFallbacks bounds the fallback receipts a drain's state keeps, the
// latest last: each request is asked of the runner once, so the receipts
// grow with the issues a drain judges, and past the bound the oldest go.
const maxJudgeFallbacks = 32

// drainJudgeTimeout bounds one routed judgement when the caller names none:
// one question over one record, read and answered, well inside the lease of
// the reservation the drain took to ask it.
const drainJudgeTimeout = 15 * time.Minute

// drainJudgeGrants are the tools the routed judge is granted and refused,
// built from its own directory dir and the checkout: it may read only dir and
// write only its answer file there, and it is refused both reading and
// writing anywhere in the checkout. Each rule names its real absolute path in
// the harness's permission-rule form (`//` anchors at the filesystem root),
// and a grant matches only where the path asked and the file it resolves to
// both match, so dir is named by its real path (judgeDir). A harness that
// edits files judges a write against the Edit rules, so the answer's grant is
// Edit's; under dontAsk anything no grant names is refused, not asked, and a
// refusal is judged before any grant.
func drainJudgeGrants(dir, checkout string) (granted, denied []string) {
	granted = []string{"Read(" + absPathRule(dir) + "/**)", "Edit(" + absPathRule(filepath.Join(dir, drainJudgeAnswerName)) + ")"}
	c := absPathRule(checkout)
	denied = []string{"Read(" + c + "/**)", "Edit(" + c + "/**)"}
	return granted, denied
}

// absPathRule is the absolute path p as a permission rule names it: `//`
// before the path, and every character the rule's pattern syntax (gitignore's)
// reads as a wildcard or an escape escaped, so the rule matches p alone.
func absPathRule(p string) string {
	var b strings.Builder
	b.WriteString("/")
	for _, r := range filepath.ToSlash(p) {
		if strings.ContainsRune(`\*?[]`, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// RoutedJudgementAnswer is the routed judge's answer, read strictly: the
// host's shape less the digest, which abcd computes itself.
type RoutedJudgementAnswer struct {
	SchemaVersion int    `json:"schema_version"`
	Issue         string `json:"issue"`
	Answer        string `json:"answer"`
	Kind          string `json:"kind,omitempty"`
	Reason        string `json:"reason"`
}

// judgeRouted reports whether the drain's judge is routed to a runner this
// call can start: a runner configuration naming one for the role, and a
// store for its transcript.
func judgeRouted(o Options, d DrainOptions) bool {
	return o.Runners != nil && d.Transcripts != nil && o.Runners.RouteFor(RoleDrainJudge).Runner != runner.Host
}

// routedJudgement asks the routed judge for the judgement j over iss and
// returns the judgement it answered, taken as a host's answer would be under
// plan. With no route it returns nil and asks nothing. When the runner did not
// answer, it returns nil with the clause the move's next step adds, and the
// host's request stands; each fallback receipt is kept in st.
func routedJudgement(repoRoot string, iss capture.Issue, v capture.DrainVerdict, j DrainJudging, plan capture.DrainPlan,
	st *DrainState, o Options, d DrainOptions, now time.Time) (*DrainJudgement, string, error) {
	if !judgeRouted(o, d) {
		return nil, "", nil
	}
	ctx := d.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return nil, "", judgeStopped(iss.ID)
	}
	dir, err := judgeDir(repoRoot)
	if err != nil {
		return nil, "the drain judge was not started for " + iss.ID + " (" + termsafe.Sanitize(fsutil.RedactHome(err.Error())) +
			"), so the request is the host's", nil
	}
	defer os.RemoveAll(dir)
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, "", fmt.Errorf("opening the drain judge's directory: %w", err)
	}
	defer root.Close()
	routed := j
	routed.Request, routed.Answer = filepath.Join(dir, drainJudgeBriefName), filepath.Join(dir, drainJudgeAnswerName)
	if err := fsutil.WriteFileAtomicInRoot(root, drainJudgeBriefName, []byte(judgementRequest(iss, v, routed, true)), filePerm); err != nil {
		return nil, "", fmt.Errorf("writing the drain judge's brief: %w", err)
	}
	timeout := d.JudgeTimeout
	if timeout == 0 {
		timeout = drainJudgeTimeout
	}
	// The runner runs in the judge's directory, may read only that and write
	// only its answer, and is refused the checkout, which is named otherwise
	// only as the place whose programs are never run.
	granted, denied := drainJudgeGrants(dir, fsutil.RealExistingPath(filepath.Clean(repoRoot)))
	req := runner.Request{
		Role:      RoleDrainJudge,
		Brief:     routed.Request,
		Receipt:   routed.Answer,
		Dir:       dir,
		Checkout:  filepath.Clean(repoRoot),
		Tools:     granted,
		Denied:    denied,
		SessionID: fmt.Sprintf("%s-%s-%d", RoleDrainJudge, iss.ID, now.Unix()),
		Timeout:   timeout,
	}
	var taken *DrainJudgement
	dsp := &runner.Dispatcher{
		Config:      o.Runners,
		HostSession: true,
		Transcripts: d.Transcripts,
		Validate: func(_ string, _ runner.Request, _ runner.Answer) error {
			out, err := readRoutedAnswer(root, j, plan, now)
			if err != nil {
				return err
			}
			taken = &out
			return nil
		},
		Record: func(fb runner.FallbackReceipt) error {
			// A runner stopped with the drain is no fallback: the
			// move refuses below, and the next one asks again.
			if ctx.Err() == nil {
				keepJudgeFallback(st, fb)
			}
			return nil
		},
		Now: func() time.Time { return now },
	}
	out, err := dsp.Dispatch(ctx, req)
	if ctx.Err() != nil {
		return nil, "", judgeStopped(iss.ID)
	}
	switch {
	case err != nil:
		return nil, "the drain judge's runner did not judge " + iss.ID + " (" + termsafe.Sanitize(fsutil.RedactHome(err.Error())) +
			"), so the request is the host's", nil
	case out.Handoff:
		why := "it handed the role back"
		if fb := out.Fallback; fb != nil {
			why = fmt.Sprintf("the %s runner was %s, recorded as a fallback", fb.Asked, fb.Reason)
		}
		return nil, "the drain judge did not judge " + iss.ID + " (" + why + "), so the request is the host's", nil
	case taken == nil:
		return nil, "the drain judge ran but handed no answer back, so the request is the host's", nil
	}
	route := out.Receipt.Route
	taken.Route = &route
	return taken, "", nil
}

// keepJudgeFallback records fb in st, keeping the latest maxJudgeFallbacks.
func keepJudgeFallback(st *DrainState, fb runner.FallbackReceipt) {
	st.JudgeFallbacks = append(st.JudgeFallbacks, fb)
	if over := len(st.JudgeFallbacks) - maxJudgeFallbacks; over > 0 {
		st.JudgeFallbacks = slices.Delete(st.JudgeFallbacks, 0, over)
	}
}

// judgeAskedClause is the move's clause for a request the routed judge was
// started on in an earlier move and did not answer: it is not started again.
func judgeAskedClause(issue string) string {
	return "the drain judge was already asked over " + issue + " for this request and did not answer, so it is not started again and the request is the host's"
}

// judgeStopped is the move's refusal when the drain was stopped while its
// judge ran over issue, or before it could start.
func judgeStopped(issue string) error {
	return refuse(StageDrain, "", "", "the drain was stopped while its judge ran over "+issue+
		", and the judge's runner was stopped with it, its process group killed; the move took no answer",
		"run `abcd drain` again, which asks the judge again")
}

// judgeDir makes the routed judge's directory: a new one, private to this
// account (os.MkdirTemp makes it 0700), in the system's temporary directory, and refused, removed
// again, when that lies inside the checkout, so the judge's directory is
// never a part of the working tree the drain and its peers read. It returns
// the directory's real path, every symbolic link resolved (the system's
// temporary directory is reached through one on macOS), since the judge's
// grants name that path and a grant holds only where the real path matches.
func judgeDir(repoRoot string) (string, error) {
	dir, err := os.MkdirTemp("", "abcd-drain-judge-")
	if err != nil {
		return "", fmt.Errorf("making a directory of its own: %w", err)
	}
	real, checkout := fsutil.RealExistingPath(dir), fsutil.RealExistingPath(filepath.Clean(repoRoot))
	if fsutil.PathWithin(real, checkout, fsutil.CaseFoldingFS()) {
		_ = os.RemoveAll(dir)
		return "", errors.New("the system's temporary directory lies inside the checkout, and the judge runs only in a directory outside it")
	}
	return real, nil
}

// readRoutedAnswer reads the routed judge's answer to j, strictly, fills in
// the digest abcd computed over the remedy the brief showed, and judges it
// as a host's answer is judged. Every refusal is the validator's, so the
// dispatch records a fallback and the host's request stands.
func readRoutedAnswer(root *os.Root, j DrainJudging, plan capture.DrainPlan, now time.Time) (DrainJudgement, error) {
	data, err := fsutil.ReadGuardedInRoot(root, drainJudgeAnswerName, maxJudgementBytes)
	if err != nil {
		return DrainJudgement{}, fmt.Errorf("the answer %s cannot be read: %v", drainJudgeAnswerName, fsutil.RedactHome(err.Error()))
	}
	var ra RoutedJudgementAnswer
	if err := jsonstrict.Decode(data, &ra); err != nil {
		return DrainJudgement{}, fmt.Errorf("the answer %s does not parse as a judgement: %s", drainJudgeAnswerName, termsafe.Sanitize(err.Error()))
	}
	a := JudgementAnswer{SchemaVersion: ra.SchemaVersion, Issue: ra.Issue, RemedySHA256: j.RemedySHA256,
		Answer: ra.Answer, Kind: ra.Kind, Reason: ra.Reason}
	if gaps := judgementGaps(a, j); len(gaps) > 0 {
		return DrainJudgement{}, fmt.Errorf("the answer %s is missing %s", drainJudgeAnswerName, strings.Join(gaps, "; "))
	}
	return judgeAnswer(a, j, plan, now)
}
