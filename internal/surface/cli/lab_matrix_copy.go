package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/matrix"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// The bounds of what the host hands back: a receipt, a copy's transcript, and
// a writing or check copy's answer.
const (
	maxCopyReceiptFile    = 64 << 10
	maxCopyTranscriptFile = 64 << 20
	maxCopyAnswerFile     = 256 << 10
	maxReviewAnswerFile   = 256 << 10
	maxVerdictFile        = 4 << 20
	maxCalibrationFile    = 256 << 10
)

// matrixHome is the account's home, which holds the clone store and the labs.
func matrixHome(verb string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", fmt.Errorf("%s: the account's home directory cannot be named: %v", verb, err)
	}
	return home, nil
}

// readHanded reads a file the host names for verb, bounded, or refuses at exit
// 2 naming it with the home directory as ~.
func readHanded(verb, what, p string, limit int64) ([]byte, error) {
	data, err := fsutil.ReadGuarded(p, limit)
	if err != nil {
		return nil, &exitError{Code: 2, Msg: verb + ": the " + what + " " + termsafe.Sanitize(fsutil.RedactHome(p)) +
			" cannot be read: " + termsafe.Sanitize(fsutil.RedactHome(err.Error())) + "; nothing was written"}
	}
	return data, nil
}

// verbFailure is matrixFailure under the sub-verb's own name.
func verbFailure(verb string, err error) error {
	e := matrixFailure(err)
	if ee, ok := e.(*exitError); ok {
		ee.Msg = strings.Replace(ee.Msg, "abcd lab matrix: ", verb+": ", 1)
	}
	return e
}

// newLabMatrixCopyCommand builds `lab matrix copy`: one handed-off copy shown,
// or handed back by the host and written as its trial.
func newLabMatrixCopyCommand(asJSON *bool) *cobra.Command {
	var receipt, transcript, answer, form string
	cmd := &cobra.Command{
		Use: "copy <lab-id> <run-id> <task>/<cell>/<repeat>",
		Long: "abcd lab matrix copy <lab-id> <run-id> <task>/<cell>/<repeat> [--receipt <file>]\n\n" +
			"Show one copy's hand-off, writing nothing: its state, its clone and the brief the\n" +
			"copy is made from, which names no task id, cell, route or model. A host cell's\n" +
			"copy is the host's to make: hand its brief to a fresh sub-agent working in the\n" +
			"clone. A runner cell's copy waits for `abcd lab matrix launch`, and the host never\n" +
			"makes it.\n\n" +
			"--receipt <file> hands a host cell's copy back: a JSON object with schema_version\n" +
			"1, outcome made or failed, a reason for a failure, and optionally\n" +
			"harness_version, model_served, started_at, finished_at, model_time_ms and\n" +
			"gate_wait_ms. A made code copy whose task names its hidden tests carries no\n" +
			"score: abcd placed them outside the clone at the run's start and runs them now,\n" +
			"in an export of the copy, so the score is abcd's run, and a score handed back is\n" +
			"refused. A launched copy whose run of them reached no outcome is handed back the\n" +
			"same way, with a made receipt alone, to run them again. A made code copy of a\n" +
			"task naming none carries score, its hidden tests' passed and total as the host\n" +
			"ran them; the trial says abcd neither placed nor ran them.\n" +
			"--transcript <file> is the copy's transcript, which the compliance check reads\n" +
			"for reads outside the clone; without it that rule is unexamined. A made writing\n" +
			"copy names its document with --answer <file>, and a check copy its answer with\n" +
			"--answer and --form (doc-fidelity or consistency). The copy is checked for\n" +
			"compliance under one overall deadline, a check copy's findings are recorded\n" +
			"before its clone is deleted, and the trial is written through the secret screen.\n\n" +
			"Refused, exit 2, with nothing written: a key that is not one, a copy the run did\n" +
			"not hand off to the host, one already handed back, a receipt, score or answer\n" +
			"that does not validate, and a trial the secret screen refuses. A check copy's\n" +
			"clone is deleted only once its trial is written. A refusal raised once a check\n" +
			"copy's findings are recorded, or a writing copy's document is kept, says so\n" +
			"instead. A halted lab exits 1.",
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			const verb = "abcd lab matrix copy"
			root, err := labRoot(verb)
			if err != nil {
				return err
			}
			home, err := matrixHome(verb)
			if err != nil {
				return err
			}
			key, err := matrix.ParseTrialKey(args[2])
			if err != nil {
				return verbFailure(verb, err)
			}
			if receipt == "" {
				for _, f := range []string{"transcript", "answer", "form"} {
					if cmd.Flags().Changed(f) {
						return &exitError{Code: 2, Msg: verb + ": --" + f + " goes with --receipt, which hands the copy back; nothing was written"}
					}
				}
				h, err := matrix.ReadHandOff(root, args[0], args[1], key)
				if err != nil {
					return matrixReportFailure(err)
				}
				h.Clone, h.Brief, h.Gates = fsutil.RedactHome(h.Clone), fsutil.RedactHome(h.Brief), nil
				return render(cmd.OutOrStdout(), *asJSON, h, func(w io.Writer) { renderHandOff(w, verb, key, h) })
			}
			back := matrix.CopyBack{Form: matrix.CheckForm(form)}
			if back.Receipt, err = readHanded(verb, "receipt", receipt, maxCopyReceiptFile); err != nil {
				return err
			}
			if transcript != "" {
				if back.Transcript, err = readHanded(verb, "transcript", transcript, maxCopyTranscriptFile); err != nil {
					return err
				}
			}
			if answer != "" {
				if back.Answer, err = readHanded(verb, "answer", answer, maxCopyAnswerFile); err != nil {
					return err
				}
			}
			t, err := matrix.CollectCopy(root, home, args[0], args[1], key, back)
			if err != nil {
				return verbFailure(verb, err)
			}
			return render(cmd.OutOrStdout(), *asJSON, t, func(w io.Writer) {
				fmt.Fprintf(w, "%s %s: %s\n  %s\n", verb, termsafe.Sanitize(key.String()), t.Status, termsafe.Sanitize(t.Reason))
				if t.Compliance != nil {
					for _, c := range t.Compliance.Checks {
						fmt.Fprintf(w, "  %-14s %-10s %s\n", c.Rule, c.State, termsafe.Sanitize(c.Detail))
					}
				}
			})
		},
	}
	cmd.Flags().StringVar(&receipt, "receipt", "", "the host's receipt of the copy it made, a JSON file; hands the copy back and writes its trial")
	cmd.Flags().StringVar(&transcript, "transcript", "", "the copy's transcript as its harness wrote it, which the compliance check reads")
	cmd.Flags().StringVar(&answer, "answer", "", "a writing copy's document, or a check copy's answer")
	cmd.Flags().StringVar(&form, "form", "", "a check copy's answer form: doc-fidelity or consistency")
	return cmd
}

// renderHandOff prints copy key's hand-off, every field read back from the
// lab's state escaped, its state among them.
func renderHandOff(w io.Writer, verb string, key matrix.TrialKey, h matrix.HandOff) {
	fmt.Fprintf(w, "%s %s: %s\n  clone: %s\n  brief: %s\n", verb, termsafe.Sanitize(key.String()), termsafe.Sanitize(string(h.State)),
		termsafe.Sanitize(cmpOr(h.Clone, "none")), termsafe.Sanitize(h.Brief))
	if h.Reason != "" {
		fmt.Fprintf(w, "  why:   %s\n", termsafe.Sanitize(h.Reason))
	}
}

// renderReviewRequests prints a code task's review requests, every field read
// back from the lab's state escaped, each request's label and state among
// them.
func renderReviewRequests(w io.Writer, verb string, out labMatrixReview) {
	fmt.Fprintf(w, "%s %s: %d request(s)\n", verb, termsafe.Sanitize(out.Task), len(out.Requests))
	for _, r := range out.Requests {
		fmt.Fprintf(w, "  %-14s %s %s (%d bytes)\n", termsafe.Sanitize(r.Reviewer), termsafe.Sanitize(r.Label), termsafe.Sanitize(string(r.State)), len(r.Request))
	}
}

// cmpOr is s, or alt where s is empty.
func cmpOr(s, alt string) string {
	if s == "" {
		return alt
	}
	return s
}

// labMatrixReview is a code task's review as the verb shows it.
type labMatrixReview struct {
	Task     string                 `json:"task"`
	Requests []matrix.ReviewHandOff `json:"requests"`
	Recorded int                    `json:"recorded,omitempty"`
}

// newLabMatrixReviewCommand builds `lab matrix review`: a code task's blinded
// reviewer requests, and with --answers the reviewers' answers recorded.
func newLabMatrixReviewCommand(asJSON *bool) *cobra.Command {
	var answers string
	cmd := &cobra.Command{
		Use: "review <lab-id> <run-id> <task>",
		Long: "abcd lab matrix review <lab-id> <run-id> <task> [--answers <dir>]\n\n" +
			"Draw, once, a blinded request for each of a code task's copies its hidden tests\n" +
			"judged, and hand every request to every reviewer cell. A request is assembled\n" +
			"from the frozen tree and the copy's clone read as data, under an opaque label,\n" +
			"with no trailer, author, branch or run metadata, and the run keeps each copy's\n" +
			"known truth apart from it. A host reviewer cell's request is the host's to\n" +
			"hand to a fresh sub-agent, which answers with the one JSON object the request\n" +
			"asks for; a runner reviewer cell's waits for a launcher. Asked again, the verb\n" +
			"shows the requests it drew.\n\n" +
			"--answers <dir> records the answers, each at <dir>/<reviewer-cell>/<label>.json,\n" +
			"read as untrusted input and scored before anything is recorded.\n\n" +
			"Refused, exit 2, with nothing written: a task that is not one of the run's code\n" +
			"tasks, a task no copy of which its hidden tests judged, an answer that does not\n" +
			"validate or names a label never asked about, an answer for a runner reviewer\n" +
			"cell, reviews already recorded, a task with a copy handed to the host and not\n" +
			"handed back, and a request the secret screen refuses. A halted lab exits 1.",
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			const verb = "abcd lab matrix review"
			root, err := labRoot(verb)
			if err != nil {
				return err
			}
			home, err := matrixHome(verb)
			if err != nil {
				return err
			}
			out := labMatrixReview{Task: args[2]}
			if answers == "" {
				if out.Requests, err = matrix.ReviewRequests(root, home, args[0], args[1], args[2]); err != nil {
					return verbFailure(verb, err)
				}
			} else {
				given, err := readReviewAnswers(verb, answers)
				if err != nil {
					return err
				}
				if out.Recorded, err = matrix.RecordReviewAnswers(root, args[0], args[1], args[2], given); err != nil {
					return verbFailure(verb, err)
				}
			}
			if out.Requests == nil {
				out.Requests = []matrix.ReviewHandOff{}
			}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				if answers != "" {
					fmt.Fprintf(w, "%s %s: %d answer(s) recorded\n", verb, termsafe.Sanitize(out.Task), out.Recorded)
					return
				}
				renderReviewRequests(w, verb, out)
			})
		},
	}
	cmd.Flags().StringVar(&answers, "answers", "", "the folder of the reviewers' answers, <dir>/<reviewer-cell>/<label>.json; records them")
	return cmd
}

// readReviewAnswers reads every answer under dir, by reviewer cell and label.
func readReviewAnswers(verb, dir string) (map[string]map[string][]byte, error) {
	out := map[string]map[string][]byte{}
	cells, err := os.ReadDir(dir)
	if err != nil {
		return nil, &exitError{Code: 2, Msg: verb + ": the answers folder " + termsafe.Sanitize(fsutil.RedactHome(dir)) + " cannot be read; nothing was written"}
	}
	for _, c := range cells {
		if !c.IsDir() {
			continue
		}
		files, err := os.ReadDir(filepath.Join(dir, c.Name()))
		if err != nil {
			return nil, &exitError{Code: 2, Msg: verb + ": the answers of " + termsafe.Sanitize(c.Name()) + " cannot be read; nothing was written"}
		}
		for _, f := range files {
			label, ok := strings.CutSuffix(f.Name(), ".json")
			if !ok || f.IsDir() {
				continue
			}
			data, err := readHanded(verb, "answer", filepath.Join(dir, c.Name(), f.Name()), maxReviewAnswerFile)
			if err != nil {
				return nil, err
			}
			if out[c.Name()] == nil {
				out[c.Name()] = map[string][]byte{}
			}
			out[c.Name()][label] = data
		}
	}
	return out, nil
}

// labMatrixJudged is a writing task's judging as the verb shows it.
type labMatrixJudged struct {
	Task      string `json:"task"`
	Packet    string `json:"packet"`
	Documents int    `json:"documents,omitempty"`
	Slots     int    `json:"slots,omitempty"`
	Members   int    `json:"members,omitempty"`
	Guests    int    `json:"guests,omitempty"`
	Host      int    `json:"host_self_blinded,omitempty"`
}

// newLabMatrixJudgeCommand builds `lab matrix judge`: a writing task's judging
// drawn from its copies, and with --verdicts a session's verdicts recorded; a
// check task's judging in two phases, its matcher's answer, the person's
// locked verdicts and the second phase's table (spec step 9).
func newLabMatrixJudgeCommand(asJSON *bool) *cobra.Command {
	var good, bad, judges, verdicts, host, matcher, proposals, labels string
	var table bool
	cmd := &cobra.Command{
		Use: "judge <lab-id> <run-id> <task>",
		Long: "abcd lab matrix judge <lab-id> <run-id> <task> --good <file> --bad <file> --judges <n>\n" +
			"abcd lab matrix judge <lab-id> <run-id> <task> --verdicts <file> [--host <login>]\n" +
			"abcd lab matrix judge <lab-id> <run-id> <check> [--matcher <file> | --verdicts <file>]\n" +
			"abcd lab matrix judge <lab-id> <run-id> <check> --table [--proposals <file>] [--labels <file>]\n\n" +
			"Draw a writing task's judging, once, from its made copies: the packet a judging\n" +
			"session is given and the key that stays on this machine, both written in the\n" +
			"run's state. Each copy is rendered as inert text with every mention of where it\n" +
			"ran, its alias, every task, cell, route and model, and every maker withheld,\n" +
			"and the packet names no run, task, cell or path. --good and --bad are the\n" +
			"examples each judge calibrates on, and --judges how many subsets of at most\n" +
			"eight copies to draw, enough that every pair of models meets in one.\n\n" +
			"--verdicts <file> records a judging session's verdict file against that\n" +
			"packet, validated as untrusted input, one ranking per judge identity; --host\n" +
			"names the host's own GitHub login, whose rankings are reported apart.\n\n" +
			"A check task is judged in two phases. Alone, the verb joins its copies'\n" +
			"findings by their exact keys and draws, once, the packet: every distinct\n" +
			"finding shuffled under a code, with no model, cell, run or path named and no\n" +
			"verdict proposed. Where the declaration names a matcher cell and two findings\n" +
			"or more are left apart, it first draws the matcher's request about those\n" +
			"alone; --matcher <file> records the matcher's answer, joins its pairs, draws a\n" +
			"sample of them for people to label, and draws the packet. With no matcher the\n" +
			"findings no key joins stay rows of their own. Asked again, the verb shows the\n" +
			"judging as it stands. --verdicts <file> locks the person's verdict on every\n" +
			"finding, once. --table builds the second phase once the verdicts are locked:\n" +
			"each finding a row and each model a column, with each model's precision,\n" +
			"unique valid findings, recall and overlap. --proposals <file> is the proposer\n" +
			"cell's verdict on each finding, given exactly when the declaration names a\n" +
			"proposer; --labels <file> is the people's labels on the matcher's sample, and\n" +
			"its agreement is reported on ten labelled pairs or more, otherwise not\n" +
			"estimable. A matcher or proposer on a runner route waits for a launcher, and\n" +
			"the host never answers for it. Nothing of a check is filed or written where\n" +
			"the release gates read.\n\n" +
			"Refused, exit 2, with no record written: a task that is not one of the run's\n" +
			"writing or check tasks, no made copy, a missing example, too few judges, a\n" +
			"judging already drawn, a task with a copy handed to the host and not handed\n" +
			"back, a packet the secret screen refuses, a verdict, matcher, proposal or label\n" +
			"file that does not validate, a second verdict file or table, a table before the\n" +
			"verdicts are locked, and an answer for a runner cell. A halted lab exits 1.",
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			const verb = "abcd lab matrix judge"
			root, err := labRoot(verb)
			if err != nil {
				return err
			}
			kind, err := matrix.RunTaskKind(root, args[0], args[1], args[2])
			if err != nil {
				return verbFailure(verb, err)
			}
			if kind == matrix.KindCheck {
				return judgeCheck(cmd, *asJSON, verb, root, args, checkJudgeFlags{matcher: matcher, verdicts: verdicts,
					proposals: proposals, labels: labels, table: table})
			}
			for _, f := range []string{"matcher", "proposals", "labels", "table"} {
				if cmd.Flags().Changed(f) {
					return &exitError{Code: 2, Msg: verb + ": --" + f + " goes with a check task's judging, and " + termsafe.Sanitize(args[2]) + " is not one of the run's check tasks; nothing was written"}
				}
			}
			out := labMatrixJudged{Task: args[2]}
			if verdicts != "" {
				for _, f := range []string{"good", "bad", "judges"} {
					if cmd.Flags().Changed(f) {
						return &exitError{Code: 2, Msg: verb + ": --" + f + " draws the judging, and --verdicts records it; nothing was written"}
					}
				}
				data, err := readHanded(verb, "verdict file", verdicts, maxVerdictFile)
				if err != nil {
					return err
				}
				res, err := matrix.RecordVerdicts(root, args[0], args[1], args[2], data, host)
				if err != nil {
					return verbFailure(verb, err)
				}
				out.Packet, out.Members, out.Guests, out.Host = res.Packet, len(res.Members), len(res.Guests), len(res.HostSelfBlinded)
				return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
					fmt.Fprintf(w, "%s %s: recorded %d member, %d guest and %d host self-blinded ranking(s)\n", verb,
						termsafe.Sanitize(out.Task), out.Members, out.Guests, out.Host)
				})
			}
			if host != "" {
				return &exitError{Code: 2, Msg: verb + ": --host goes with --verdicts; nothing was written"}
			}
			n, err := strconv.Atoi(judges)
			if err != nil || good == "" || bad == "" {
				return &exitError{Code: 2, Msg: verb + ": drawing the judging takes --good <file>, --bad <file> and --judges <n>; nothing was written"}
			}
			home, err := matrixHome(verb)
			if err != nil {
				return err
			}
			g, err := readHanded(verb, "good example", good, maxCalibrationFile)
			if err != nil {
				return err
			}
			b, err := readHanded(verb, "bad example", bad, maxCalibrationFile)
			if err != nil {
				return err
			}
			p, k, err := matrix.DrawWriting(root, home, args[0], args[1], args[2], string(g), string(b), n)
			if err != nil {
				return verbFailure(verb, err)
			}
			out.Packet, out.Documents, out.Slots = k.Packet, len(p.Documents), len(p.Slots)
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				fmt.Fprintf(w, "%s %s: drew %d document(s) in %d subset(s); packet %s\n", verb, termsafe.Sanitize(out.Task),
					out.Documents, out.Slots, out.Packet)
			})
		},
	}
	cmd.Flags().StringVar(&good, "good", "", "the good example each judge calibrates on, a file")
	cmd.Flags().StringVar(&bad, "bad", "", "the bad example each judge calibrates on, a file")
	cmd.Flags().StringVar(&judges, "judges", "", "how many judges' subsets to draw, a whole number")
	cmd.Flags().StringVar(&verdicts, "verdicts", "", "a judging session's verdict file; records it")
	cmd.Flags().StringVar(&host, "host", "", "with --verdicts: the host's own GitHub login, whose rankings are reported apart")
	cmd.Flags().StringVar(&matcher, "matcher", "", "a check task's matcher answer, a JSON file; records it and draws the packet")
	cmd.Flags().BoolVar(&table, "table", false, "builds a check task's second phase, once its verdicts are locked")
	cmd.Flags().StringVar(&proposals, "proposals", "", "with --table: the proposer cell's verdict on each finding, a JSON file")
	cmd.Flags().StringVar(&labels, "labels", "", "with --table: the people's labels on the matcher's sample, a JSON file")
	return cmd
}

// checkJudgeFlags is what `lab matrix judge` was given for a check task.
type checkJudgeFlags struct {
	matcher, verdicts, proposals, labels string
	table                                bool
}

// maxMatcherFile bounds a matcher's answer, a proposal file and a label file
// as read from the host; the core holds each to its own bound after.
const maxMatcherFile = 4 << 20

// labMatrixCheckLocked is a check's locked verdicts as the verb shows them.
type labMatrixCheckLocked struct {
	Task   string `json:"task"`
	Packet string `json:"packet"`
	Judged int    `json:"judged"`
	Valid  int    `json:"valid"`
}

// judgeCheck is `lab matrix judge` on a check task: one step of its judging,
// or, given none, the draw of its first phase or the judging as it stands.
func judgeCheck(cmd *cobra.Command, asJSON bool, verb, root string, args []string, f checkJudgeFlags) error {
	for _, flag := range []string{"good", "bad", "judges", "host"} {
		if cmd.Flags().Changed(flag) {
			return &exitError{Code: 2, Msg: verb + ": --" + flag + " goes with a writing task's judging, and " + termsafe.Sanitize(args[2]) + " is a check; nothing was written"}
		}
	}
	steps := 0
	for _, on := range []bool{f.matcher != "", f.verdicts != "", f.table} {
		if on {
			steps++
		}
	}
	if steps > 1 {
		return &exitError{Code: 2, Msg: verb + ": --matcher, --verdicts and --table are one step each; give one at a time; nothing was written"}
	}
	if !f.table && (f.proposals != "" || f.labels != "") {
		return &exitError{Code: 2, Msg: verb + ": --proposals and --labels go with --table; nothing was written"}
	}
	w := cmd.OutOrStdout()
	lab, run, task := args[0], args[1], args[2]
	switch {
	case f.matcher != "":
		data, err := readHanded(verb, "matcher's answer", f.matcher, maxMatcherFile)
		if err != nil {
			return err
		}
		j, err := matrix.RecordCheckMatcher(root, lab, run, task, data)
		if err != nil {
			return verbFailure(verb, err)
		}
		return render(w, asJSON, j, func(w io.Writer) { renderCheckJudging(w, verb, j) })
	case f.verdicts != "":
		data, err := readHanded(verb, "verdict file", f.verdicts, maxVerdictFile)
		if err != nil {
			return err
		}
		v, err := matrix.RecordCheckVerdicts(root, lab, run, task, data)
		if err != nil {
			return verbFailure(verb, err)
		}
		out := labMatrixCheckLocked{Task: task, Packet: v.Packet, Judged: len(v.Valid)}
		for _, ok := range v.Valid {
			if ok {
				out.Valid++
			}
		}
		return render(w, asJSON, out, func(w io.Writer) {
			fmt.Fprintf(w, "%s %s: locked %d verdict(s), %d valid; packet %s\n", verb, termsafe.Sanitize(task), out.Judged, out.Valid, termsafe.Sanitize(out.Packet))
		})
	case f.table:
		var proposals, labels []byte
		var err error
		if f.proposals != "" {
			if proposals, err = readHanded(verb, "proposals", f.proposals, maxMatcherFile); err != nil {
				return err
			}
		}
		if f.labels != "" {
			if labels, err = readHanded(verb, "labels", f.labels, maxMatcherFile); err != nil {
				return err
			}
		}
		t, err := matrix.RecordCheckTable(root, lab, run, task, proposals, labels)
		if err != nil {
			return verbFailure(verb, err)
		}
		return render(w, asJSON, t, func(w io.Writer) { renderCheckTable(w, verb, t) })
	}
	j, err := matrix.DrawCheck(root, lab, run, task)
	if err != nil {
		return verbFailure(verb, err)
	}
	return render(w, asJSON, j, func(w io.Writer) { renderCheckJudging(w, verb, j) })
}

// judgeCellLine names a check's matcher or proposer for the person.
func judgeCellLine(c *matrix.JudgeCell) string {
	if c == nil {
		return "none declared"
	}
	s := termsafe.Sanitize(c.Cell) + " (" + termsafe.Sanitize(string(c.Source)) + ")"
	if c.AwaitingRunner {
		s += ", a runner cell: its answer waits for a launcher this build does not have"
	}
	return s
}

// renderCheckJudging prints where a check task's judging stands and what is
// to be handed out next, every field read back from the lab's state escaped.
func renderCheckJudging(w io.Writer, verb string, j matrix.CheckJudging) {
	fmt.Fprintf(w, "%s %s: %s; %d copies, %d finding(s) left apart by the exact keys\n", verb, termsafe.Sanitize(j.Task),
		termsafe.Sanitize(string(j.Stage)), j.Copies, j.Leftovers)
	fmt.Fprintf(w, "  matcher:  %s\n  proposer: %s\n", judgeCellLine(j.Matcher), judgeCellLine(j.Proposer))
	if j.Waiting != "" {
		fmt.Fprintf(w, "  waiting: %s\n", termsafe.SanitizeLine(j.Waiting))
	}
	if j.MatcherRequest != nil && j.Stage == matrix.CheckAwaitingMatcher && j.Waiting == "" {
		fmt.Fprintf(w, "  the matcher's request holds %d finding(s) (--json shows it); record its answer with --matcher <file>\n", len(j.MatcherRequest.Items))
	}
	if len(j.Sample) > 0 {
		fmt.Fprintf(w, "  %d pair(s) of the matcher's are drawn for people to label; give the labels with --table --labels <file>\n", len(j.Sample))
	}
	if j.Packet != nil {
		fmt.Fprintf(w, "  packet %s: %d finding(s) to judge (--json shows it)\n", termsafe.Sanitize(j.PacketDigest), len(j.Packet.Items))
	}
	switch j.Stage {
	case matrix.CheckAwaitingVerdicts:
		fmt.Fprintln(w, "  record the person's locked verdicts with --verdicts <file>")
	case matrix.CheckVerdictsLocked:
		fmt.Fprintln(w, "  the verdicts are locked; build the second phase with --table")
	}
}

// renderCheckTable prints a check's second phase, every field escaped.
func renderCheckTable(w io.Writer, verb string, t matrix.CheckTable) {
	fmt.Fprintf(w, "%s %s: %d distinct finding(s) over %d model(s)\n", verb, termsafe.Sanitize(t.Task), len(t.Rows), len(t.Cells))
	rate := func(r matrix.Rate) string {
		if !r.Estimable() {
			return "not estimable"
		}
		return fmt.Sprintf("%d of %d", r.Hits, r.Of)
	}
	for _, s := range t.Scores {
		fmt.Fprintf(w, "  %-14s precision %s, unique valid %d, recall against the valid findings %s\n", termsafe.Sanitize(s.Cell),
			rate(s.Precision), s.UniqueValid, rate(s.RecallValid))
	}
	for _, o := range t.Overlap {
		fmt.Fprintf(w, "  overlap %s and %s: %d of %d\n", termsafe.Sanitize(o.A), termsafe.Sanitize(o.B), o.Both, o.Either)
	}
	if a := t.Matcher; a != nil {
		if a.NotEstimable != "" {
			fmt.Fprintf(w, "  matcher's agreement: not estimable: %s\n", termsafe.Sanitize(a.NotEstimable))
		} else {
			fmt.Fprintf(w, "  matcher's agreement: %s labelled pairs\n", rate(a.Agreed))
		}
	}
}
