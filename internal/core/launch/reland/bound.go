// Package reland relands a bot's dependency pull request as the person's own
// commit (iss-2611220952509874): the hand step the dependency-reauthor
// workflow leaves to a person while its owner or its App is unset.
//
// The bound is the workflow's own. The pull request is judged by the
// scaffolded dependency-reauthor script, run in its `check` mode against the
// declaration origin's default branch stores, so the bot, the branch prefix,
// the one commit and the declared manifests are decided by one parser and one
// check, never a second copy here. What passes is applied onto a fresh branch
// from origin's default branch, read as stored, committed under the checkout's
// own git identity, and left local: the person lands it with `abcd land`.
package reland

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/launch/scaffold"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// Row is one ecosystem row of the declaration, as the script parsed it.
type Row struct {
	Bot       string   `json:"bot"`
	Prefix    string   `json:"prefix"`
	Directory string   `json:"directory"`
	Files     []string `json:"files"`
}

// Declaration is the re-authoring declaration as the script parsed it:
// whether its owner is set, and its rows.
type Declaration struct {
	OwnerSet bool  `json:"owner_set"`
	Rows     []Row `json:"rows"`
}

// Matches reports the row a pull request by author from branch head falls
// under: the author is the row's bot and the branch starts with its prefix.
// It is the listing's first cut, never the bound: the script judges that.
func (d Declaration) Matches(author, head string) (Row, bool) {
	for _, r := range d.Rows {
		if r.Bot == author && strings.HasPrefix(head, r.Prefix) {
			return r, true
		}
	}
	return Row{}, false
}

// ErrNoDeclaration is what a read of a checkout without the declaration
// answers: the repository has not opted in to re-authoring.
var ErrNoDeclaration = errors.New("the repository declares no dependency re-authoring (" + scaffold.ReauthorConfPath + ")")

// readDeclaration parses the declaration in the checkout at root's working
// tree through the script's rows mode. A checkout without one answers
// ErrNoDeclaration; a malformed one, the script's refusal.
func readDeclaration(root string) (Declaration, error) {
	conf := filepath.Join(root, filepath.FromSlash(scaffold.ReauthorConfPath))
	if fi, err := os.Lstat(conf); err != nil || !fi.Mode().IsRegular() {
		return Declaration{}, ErrNoDeclaration
	}
	return declarationAt(root, conf)
}

// declarationAt runs the script's rows mode over the declaration file conf.
func declarationAt(root, conf string) (Declaration, error) {
	out, err := runScript(root, "rows", map[string]string{"REAUTHOR_CONF": conf})
	if err != nil {
		return Declaration{}, err
	}
	var d Declaration
	for _, line := range strings.Split(out, "\n") {
		switch {
		case line == "owner declared":
			d.OwnerSet = true
		case line == "owner unset", line == "":
		case strings.HasPrefix(line, "row "):
			f := strings.Fields(strings.TrimPrefix(line, "row "))
			if len(f) < 4 {
				return Declaration{}, fmt.Errorf("the re-authoring script printed a row it should have refused: %q", line)
			}
			d.Rows = append(d.Rows, Row{Bot: f[0], Prefix: f[1], Directory: f[2], Files: f[3:]})
		default:
			return Declaration{}, fmt.Errorf("the re-authoring script's rows mode printed an unknown line: %q", line)
		}
	}
	return d, nil
}

// scriptOutputCap bounds what the script may print: its output is a few
// lines, and an unbounded read is the shape a hostile declaration exploits.
const scriptOutputCap = 1 << 16

// boundRefusal is a script answer that refused to judge the pull request (a
// malformed declaration), as opposed to one that could not run at all.
type boundRefusal struct{ msg string }

func (e *boundRefusal) Error() string { return e.msg }

// runScript runs the embedded dependency-reauthor script in mode from root,
// with vars over an environment that reads history as stored and carries no
// step summary to append to. A non-zero exit is an error carrying the
// script's refusal; stdout is returned otherwise.
func runScript(root, mode string, vars map[string]string) (string, error) {
	rendered, err := scaffold.Render(scaffold.AbcdSubstitutions())
	if err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp("", "abcd-reland-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	script := filepath.Join(dir, "dependency-reauthor.sh")
	if err := os.WriteFile(script, rendered.ReauthorScript, 0o600); err != nil {
		return "", err
	}
	env := []string{}
	for _, kv := range gitutil.ScrubbedEnv() {
		key, _, _ := strings.Cut(kv, "=")
		switch key {
		case "GITHUB_STEP_SUMMARY", "REAUTHOR_CONF", "REAUTHOR_WORKFLOW", "REAUTHOR_REMOTE",
			"PR_AUTHOR", "ACTOR", "HEAD_REF", "HEAD_SHA", "BASE_SHA", "HEAD_REPO", "GITHUB_REPOSITORY",
			"DEPENDENCY_REAUTHOR_APP_ID", "DEPENDENCY_REAUTHOR_APP_KEY":
			continue
		}
		env = append(env, kv)
	}
	for k, v := range vars {
		env = append(env, k+"="+v)
	}
	cmd := exec.Command("bash", script, mode)
	cmd.Dir = root
	cmd.Env = gitutil.AsStoredEnv(env)
	var stdout, stderr capped
	stdout.limit, stderr.limit = scriptOutputCap, scriptOutputCap
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	var ee *exec.ExitError
	switch {
	case err == nil:
		return stdout.String(), nil
	case errors.As(err, &ee) && ee.ExitCode() == 1:
		msg := strings.TrimSpace(stderr.String())
		msg = strings.TrimPrefix(msg, "::error::")
		msg = strings.TrimPrefix(msg, "dependency-reauthor: ")
		return "", &boundRefusal{msg: "the re-authoring script " + msg}
	default:
		return "", fmt.Errorf("the re-authoring script could not run: %v (%s)", err, strings.TrimSpace(stderr.String()))
	}
}

// capped keeps the first limit bytes written to it and drops the rest.
type capped struct {
	bytes.Buffer
	limit int
}

func (c *capped) Write(p []byte) (int, error) {
	if room := c.limit - c.Len(); room > 0 {
		c.Buffer.Write(p[:min(room, len(p))])
	}
	return len(p), nil
}
