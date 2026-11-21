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
			"clone. A runner cell's copy waits for a launcher, and the host never makes it.\n\n" +
			"--receipt <file> hands a host cell's copy back: a JSON object with schema_version\n" +
			"1, outcome made or failed, a reason for a failure, and optionally\n" +
			"harness_version, model_served, started_at, finished_at, model_time_ms and\n" +
			"gate_wait_ms. A made code copy carries score, its hidden tests' passed and total\n" +
			"as the host ran them; the trial says abcd neither placed nor ran them.\n" +
			"--transcript <file> is the copy's transcript, which the compliance check reads\n" +
			"for reads outside the clone; without it that rule is unexamined. A made writing\n" +
			"copy names its document with --answer <file>, and a check copy its answer with\n" +
			"--answer and --form (doc-fidelity or consistency). The copy is checked for\n" +
			"compliance under one overall deadline, a check copy's findings are recorded\n" +
			"before its clone is deleted, and the trial is written through the secret screen.\n\n" +
			"Refused, exit 2, with nothing written: a key that is not one, a copy the run did\n" +
			"not hand off to the host, one already handed back, a receipt, score or answer\n" +
			"that does not validate, and a trial the secret screen refuses. A check copy's\n" +
			"clone is deleted only once its trial is written. A halted lab exits 1.",
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
				return render(cmd.OutOrStdout(), *asJSON, h, func(w io.Writer) {
					fmt.Fprintf(w, "%s %s: %s\n  clone: %s\n  brief: %s\n", verb, termsafe.Sanitize(key.String()), h.State,
						termsafe.Sanitize(cmpOr(h.Clone, "none")), termsafe.Sanitize(h.Brief))
					if h.Reason != "" {
						fmt.Fprintf(w, "  why:   %s\n", termsafe.Sanitize(h.Reason))
					}
				})
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
				fmt.Fprintf(w, "%s %s: %d request(s)\n", verb, termsafe.Sanitize(out.Task), len(out.Requests))
				for _, r := range out.Requests {
					fmt.Fprintf(w, "  %-14s %s %s (%d bytes)\n", termsafe.Sanitize(r.Reviewer), r.Label, r.State, len(r.Request))
				}
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
// drawn from its copies, and with --verdicts a session's verdicts recorded.
func newLabMatrixJudgeCommand(asJSON *bool) *cobra.Command {
	var good, bad, judges, verdicts, host string
	cmd := &cobra.Command{
		Use: "judge <lab-id> <run-id> <task>",
		Long: "abcd lab matrix judge <lab-id> <run-id> <task> --good <file> --bad <file> --judges <n>\n" +
			"abcd lab matrix judge <lab-id> <run-id> <task> --verdicts <file> [--host <login>]\n\n" +
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
			"Refused, exit 2, with nothing written: a task that is not one of the run's\n" +
			"writing tasks, no made copy, a missing example, too few judges, a judging\n" +
			"already drawn, a task with a copy handed to the host and not handed back, a\n" +
			"packet the secret screen refuses, and a verdict file that does not validate.\n" +
			"A halted lab exits 1.",
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			const verb = "abcd lab matrix judge"
			root, err := labRoot(verb)
			if err != nil {
				return err
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
	return cmd
}
