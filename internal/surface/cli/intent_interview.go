package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/interview"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// planningTools are the tools the planning-interviewer's contract grants in a
// plain Terminal: it reads the record and edits it, as the host agent does
// (open question 4, decided (a)); Write is added by the loop for its receipt.
var planningTools = []string{"Read", "Edit", "Grep", "Glob"}

// planningSeed is what the planning interview opens from: the intent, where
// its record is, its folder, and the planning brief the pre-pass wrote, when
// there is one.
type planningSeed struct {
	Intent        string `json:"intent"`
	Path          string `json:"path"`
	Bucket        string `json:"bucket"`
	PlanningBrief string `json:"planning_brief,omitempty"`
}

// planningOutcome is the planning interview's done: what the interview
// changed in the record.
type planningOutcome struct {
	Summary string `json:"summary"`
}

func parsePlanningOutcome(raw json.RawMessage) (planningOutcome, error) {
	var o planningOutcome
	if err := jsonstrict.Decode(raw, &o); err != nil {
		return o, err
	}
	if strings.TrimSpace(o.Summary) == "" {
		return o, errors.New("the summary is empty")
	}
	return o, nil
}

// planningInterviewResult is the verb's result: the record the role edited,
// what it says it changed, the answers record, and the readiness gate on the
// record as the interview left it.
type planningInterviewResult struct {
	Intent  string             `json:"intent"`
	Path    string             `json:"path"`
	Summary string             `json:"summary"`
	Record  string             `json:"answers_record"`
	Ready   intent.ReadyResult `json:"ready"`
}

// newIntentInterviewCommand is `abcd intent interview <itd-N>`: the planning
// interview in a plain Terminal (spc-2610030911534855). The
// planning-interviewer, on the runner the person routed it to, writes each
// question and edits the record with its contract's tools after each answer,
// as the host agent does; abcd draws each question and records each answer,
// then reports the readiness gate on the record. The plan act stays the
// product thinker's, at the command line.
func newIntentInterviewCommand(asJSON *bool) *cobra.Command {
	var flags writtenFlags
	cmd := &cobra.Command{
		Use: "interview <itd-N> [--answers <file>] [--answered-in <place>]",
		Long: "Run the planning interview in a plain Terminal, with no host session: the planning-interviewer,\n" +
			"on the runner the person routed it to in their own machine's config, writes each question and edits\n" +
			"the intent record with the tools its contract grants; abcd draws each question, takes the answer,\n" +
			"and reports the readiness gate when the interview ends. With no route of the person's to a runner it\n" +
			"refuses before anything runs, writing nothing. Off a terminal the questions are answered from\n" +
			"--answers, by ordinal (Q1, Q2, ...). The plan act stays the product thinker's: `abcd intent plan`.",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &exitError{Code: 2, Msg: "abcd intent interview: one intent is expected — abcd intent interview <itd-N>"}
			}
			if !recordid.ValidIntentID(args[0]) {
				return &exitError{Code: 2, Msg: fmt.Sprintf("abcd intent interview: %q is not an intent id (itd-N) (nothing was run)", termsafe.Sanitize(args[0]))}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := intentStoreRoot(cmd)
			if err != nil {
				return err
			}
			id := args[0]
			corpus, err := intent.Load(repoRoot)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd intent interview: " + termsafe.Sanitize(scrubPaths(err)) + " (nothing was run)"}
			}
			it, ok := corpus.Lookup(id)
			if !ok {
				return &exitError{Code: 2, Msg: fmt.Sprintf("abcd intent interview: %s is not in any folder of the intent store (nothing was run)", id)}
			}
			if it.Bucket != intent.BucketDrafts && it.Bucket != intent.BucketPlanned {
				return &exitError{Code: 2, Msg: fmt.Sprintf("abcd intent interview: %s is %s; the planning interview runs on a draft or a planned intent (nothing was run)",
					id, termsafe.Sanitize(it.Bucket))}
			}
			seed := planningSeed{Intent: id, Path: it.Path, Bucket: it.Bucket}
			brief := path.Join(intent.PlanningBriefsRelDir, id+".md")
			if fi, err := os.Lstat(filepath.Join(repoRoot, filepath.FromSlash(brief))); err == nil && fi.Mode().IsRegular() {
				seed.PlanningBrief = brief
			}
			seedJSON, err := json.Marshal(seed)
			if err != nil {
				return err
			}
			var outcome *planningOutcome
			w := &interview.Written{
				Name: interview.Planning, Verb: "abcd intent interview", Role: interview.RolePlanningInterviewer,
				Target: id, Repo: repoRoot, Task: interview.PlanningTask, Done: interview.PlanningDone,
				Seed: seedJSON, Tools: planningTools,
				CheckDone: func(raw json.RawMessage) error {
					_, err := parsePlanningOutcome(raw)
					return err
				},
				Finish: func(raw json.RawMessage) (string, error) {
					o, err := parsePlanningOutcome(raw)
					if err != nil {
						return "", &finishError{err}
					}
					outcome = &o
					return "", nil
				},
			}
			res, err := runWrittenInterview(cmd, w, flags)
			if err != nil {
				var fin *finishError
				if errors.As(err, &fin) {
					return &exitError{Code: 1, Msg: "abcd intent interview: " + termsafe.Sanitize(fin.err.Error())}
				}
				return err
			}
			if outcome == nil {
				return &exitError{Code: 1, Msg: "abcd intent interview: the interview ended without an outcome"}
			}
			// abcd's validation after the role's edits is the gate the host
			// path has: the readiness report on the record as it now stands.
			ready, err := intent.Ready(repoRoot, id)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd intent interview: the record no longer reads after the interview: " + termsafe.Sanitize(scrubPaths(err))}
			}
			out := planningInterviewResult{Intent: id, Path: ready.Path, Summary: outcome.Summary, Ready: ready}
			if res.Record != "" {
				out.Record = path.Join(interview.RecordsRel, filepath.Base(res.Record))
			}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				verdict := "READY"
				if !ready.Ready {
					verdict = "NOT READY"
				}
				fmt.Fprintf(w, "abcd intent interview — %s: the interview ended; the record is %s (%s)\n", id, verdict, termsafe.Sanitize(ready.Bucket))
				fmt.Fprintf(w, "  changed: %s\n", termsafe.Sanitize(outcome.Summary))
				for _, c := range ready.Checks {
					if !c.OK {
						fmt.Fprintf(w, "  %s: %s\n", c.Name, termsafe.Sanitize(c.Detail))
					}
				}
				if ready.Bucket == intent.BucketDrafts {
					fmt.Fprintf(w, "  next: read the changes in %s; the sign-off is the product thinker's: abcd intent plan %s\n", termsafe.Sanitize(ready.Path), id)
				} else {
					fmt.Fprintf(w, "  next: read the changes in %s\n", termsafe.Sanitize(ready.Path))
				}
			})
		},
	}
	flags.register(cmd, interview.Planning)
	return cmd
}
