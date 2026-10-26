package landing

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/assistedby"
	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/core/issuemention"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The pull-request body land composes when it finds no pull request to queue
// (iss-220). land still opens none: it writes the body to the checkout's local
// tier and names it in the create command it prints, so the body the person
// posts carries what the hosted checks ask of it. A body typed by hand was the
// one CI refused for a missing Assisted-by line, after the push.
//
// The body is the branch's own record: the subject of each commit, then each
// commit's Refs:, Resolves: and Delivers: lines and its Assisted-by: lines, once
// each, in the order the commits give them. It carries no commit's prose. It is
// judged with the checks the shell guard applies to a posted body: the
// outbound policy, the disclosure (assistedby.JudgeBody) and, where the
// repository opted in, the mentions (issuemention); a body failing one is still
// written, and the stop says which check it fails.

// bodyRelDir is where the body is written, in the checkout's local tier.
const bodyRelDir = ".abcd/.work.local/scratch/land"

// maxComposedCommits bounds the commits a body is composed from: a branch that
// far ahead is not one pull request's worth, and its body is not composed.
const maxComposedCommits = 200

var (
	declareLineRe  = regexp.MustCompile(`^(Resolves|Refs):[ \t]+iss-[0-9]+([ \t]*,[ \t]*iss-[0-9]+)*[ \t]*$`)
	deliversLineRe = regexp.MustCompile(`^Delivers:[ \t]+itd-[0-9]+([ \t]*,[ \t]*itd-[0-9]+)*[ \t]*$`)
	unsafeNameRe   = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
)

// composedBody is a body land composed: the title and body text, and the
// commits it was read from.
type composedBody struct {
	title   string
	body    string
	commits int
}

// composeBody reads the commits tracking..head in the checkout at dir, oldest
// first, merges left out.
func composeBody(dir, tracking, head string) (composedBody, error) {
	out, err := gitutil.Run(dir, "log", "--reverse", "--no-merges", fmt.Sprintf("--max-count=%d", maxComposedCommits+1),
		"--format=%x1e%s%x1f%B", tracking+".."+head, "--")
	if err != nil {
		return composedBody{}, err
	}
	var subjects, declared, assisted []string
	// allNone holds while every commit declares Assisted-by: None and names no
	// model: one commit that says nothing makes None a claim nobody made.
	allNone := true
	for _, rec := range strings.Split(out, "\x1e") {
		subject, msg, ok := strings.Cut(rec, "\x1f")
		subject = strings.TrimSpace(subject)
		if !ok || subject == "" {
			continue
		}
		subjects = append(subjects, subject)
		helped, none := false, false
		for _, l := range strings.Split(strings.ReplaceAll(msg, "\r", ""), "\n") {
			switch v, isTrailer := strings.CutPrefix(l, "Assisted-by: "); {
			case declareLineRe.MatchString(l), deliversLineRe.MatchString(l):
				declared = appendOnce(declared, l)
			case isTrailer && (assistedby.IsModelValue(v) || assistedby.IsComposed(v)):
				assisted = appendOnce(assisted, l)
				helped = true
			case isTrailer && (v == "None" || v == "none"):
				none = true
			}
		}
		if helped || !none {
			allNone = false
		}
	}
	if len(subjects) == 0 {
		return composedBody{}, fmt.Errorf("no commit between %s and %s to compose a body from", tracking, short(head))
	}
	if len(subjects) > maxComposedCommits {
		return composedBody{}, fmt.Errorf("the branch is more than %d commits ahead of %s", maxComposedCommits, tracking)
	}
	var b strings.Builder
	for _, s := range subjects {
		b.WriteString("- " + s + "\n")
	}
	trailers := declared
	switch {
	case len(assisted) > 0:
		trailers = append(trailers, assisted...)
	case allNone:
		trailers = append(trailers, "Assisted-by: None")
	}
	if len(trailers) > 0 {
		b.WriteString("\n" + strings.Join(trailers, "\n") + "\n")
	}
	return composedBody{title: subjects[0], body: b.String(), commits: len(subjects)}, nil
}

func appendOnce(list []string, s string) []string {
	for _, x := range list {
		if x == s {
			return list
		}
	}
	return append(list, s)
}

// judgeBody names the first check of the shell guard's that body fails, or ""
// when it passes them all. dir's own configuration decides the mention check.
func judgeBody(dir string, c composedBody) string {
	if findings, err := scanner.CheckOutbound(dir, c.body, "the composed pull-request body"); len(findings) > 0 {
		return "the outbound check (it carries a live agent-session URL or a tool attribution footer)"
	} else if err != nil {
		return "the outbound check, which could not run (" + err.Error() + ")"
	}
	if fault := assistedby.JudgeBody(c.body); fault != assistedby.BodyClean {
		return "the disclosure check: it carries " + fault.String() + ", since the branch's commits disclose none; " +
			"add an Assisted-by line before posting"
	}
	if guard.ForgeMentionsOptedIn(dir) {
		if ids := issuemention.Undeclared(c.title, c.body); len(ids) > 0 {
			return "the mention check: it names " + strings.Join(ids, ", ") + " without a Resolves: or Refs: line"
		}
	}
	return ""
}

// writeBody writes the composed body to the checkout's local tier and returns
// its path, or "" with the reason it wrote none: a tier git does not ignore
// would leave the checkout dirty, and the next landing would refuse it. land
// makes no tier: a checkout without one (a new worktree whose receipt another
// worktree minted) keeps nothing of land's, the release hold's sightings
// included, and gets the bare create command.
func writeBody(dir, branch, body string) (string, string) {
	rel := path.Join(bodyRelDir, "pr-body-"+strings.Trim(unsafeNameRe.ReplaceAllString(branch, "-"), "-")+".md")
	if err := gitutil.WorkingTreeCommand(dir, "check-ignore", "-q", "--", rel).Run(); err != nil {
		return "", "the local tier (" + bodyRelDir + ") is not ignored by git in this checkout, so land wrote no body file"
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", "the body file could not be written (" + err.Error() + ")"
	}
	defer root.Close()
	if fi, err := root.Lstat(TierRelDir); errors.Is(err, fs.ErrNotExist) || (err == nil && !fi.IsDir()) {
		return "", "this checkout has no local tier (" + TierRelDir + "), and land makes none, so it wrote no body file"
	} else if err != nil {
		return "", "the local tier could not be read (" + err.Error() + "), so land wrote no body file"
	}
	if err := fsutil.WriteFileAtomicInRoot(root, rel, []byte(body), 0o644); err != nil {
		return "", "the body file could not be written (" + err.Error() + ")"
	}
	return filepath.Join(dir, filepath.FromSlash(rel)), ""
}

// createStop is the stop when the branch has no pull request: the create
// command, with the composed body named where land could write one.
func createStop(res *Result, dir, tracking string) {
	base := "gh pr create --base " + shellQuote(res.Default) + " --head " + shellQuote(res.Branch)
	note := "no open pull request for " + res.Branch + "; land opens none: open it with the command, then run abcd land again to queue it"
	c, err := composeBody(dir, tracking, res.Head)
	if err != nil {
		res.add("enqueue", StatusStopped, note+"; no body was composed ("+err.Error()+")", base)
		return
	}
	file, why := writeBody(dir, res.Branch, c.body)
	if file == "" {
		res.add("enqueue", StatusStopped, note+"; "+why, base)
		return
	}
	note += fmt.Sprintf("; the body was composed from the branch's %d commit(s) into %s", c.commits, file)
	if fails := judgeBody(dir, c); fails != "" {
		note += "; it fails " + fails
	}
	res.add("enqueue", StatusStopped, note, base+" --title "+shellQuote(c.title)+" --body-file "+shellQuote(file))
}
