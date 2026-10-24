package identity

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// Proposal is the human identity the gate offers when the effective one
// diverges, and where it was read from.
type Proposal struct {
	Identity Effective
	// From names the source for the person being asked.
	From string
}

// Propose returns the identity to offer when the effective one diverges: the
// person's own record on this machine, else their global git identity
// (user.name/user.email in the caller's global config). The chain is disk-only
// by decision (itd-131, adr-38): no network lookup in an implicit path. A
// machine identity is never proposed, and nothing a repository commits is ever
// read as the proposal (itd-2610240918547728): a contributor is never offered
// anyone's identity but their own. ok is false when neither source holds a
// complete human identity.
func Propose(root string) (Proposal, bool, error) {
	return ProposeWithHome(root, UserHome())
}

// ProposeWithHome is Propose for the person whose home directory is home.
func ProposeWithHome(root, home string) (Proposal, bool, error) {
	rootSHA, err := gitutil.RootCommit(root)
	if err != nil {
		return Proposal{}, false, err
	}
	rec, ref, recorded, err := LoadRecord(home, rootSHA)
	if err != nil {
		return Proposal{}, false, err
	}
	if recorded {
		p := Proposal{Identity: Effective{Name: rec.Name, Email: rec.Email}, From: "the identity you recorded in " + ref.Display}
		return p, !IsToolIdentity(RoleAuthor, rec.Name, rec.Email), nil
	}
	return proposeGlobal(root)
}

// ProposeRecord returns the identity to offer a person who has not recorded
// who they commit as on this machine: their usual git identity, which is the
// global user.name/user.email, else the identity git resolves in this
// repository when that is a complete human one. A machine identity is never
// proposed. ok is false when no human identity can be read from disk.
func ProposeRecord(root string) (Proposal, bool, error) {
	p, ok, err := proposeGlobal(root)
	if err != nil || ok {
		return p, ok, err
	}
	eff, err := EffectiveIdentity(root)
	if err != nil {
		return Proposal{}, false, err
	}
	if eff.Name == "" || eff.Email == "" || IsToolIdentity(RoleAuthor, eff.Name, eff.Email) {
		return Proposal{}, false, nil
	}
	return Proposal{Identity: eff, From: "the git identity this repository commits under"}, true, nil
}

// proposeGlobal is the global git identity, when it is a complete human one.
func proposeGlobal(root string) (Proposal, bool, error) {
	name, err := gitConfigScope(root, "--global", "user.name")
	if err != nil {
		return Proposal{}, false, err
	}
	email, err := gitConfigScope(root, "--global", "user.email")
	if err != nil {
		return Proposal{}, false, err
	}
	if name == "" || email == "" || IsToolIdentity(RoleAuthor, name, email) {
		return Proposal{}, false, nil
	}
	return Proposal{Identity: Effective{Name: name, Email: email}, From: "your global git identity"}, true, nil
}

// Outranking names every source that outranks a repository's user.name and
// user.email for either role and holds a value other than want: a
// GIT_AUTHOR_* / GIT_COMMITTER_* environment override, or an author.* /
// committer.* config key. Writing repo-local user.* cannot change what such a
// source stamps, so a caller that would write it asks this first and, when the
// list is not empty, says what to unset instead of writing to no effect.
func Outranking(root string, want Effective) ([]string, error) {
	var out []string
	for _, role := range []Role{RoleAuthor, RoleCommitter} {
		for _, f := range []struct{ field, want string }{{"name", want.Name}, {"email", want.Email}} {
			env := "GIT_" + strings.ToUpper(string(role)) + "_" + strings.ToUpper(f.field)
			if v := strings.TrimSpace(os.Getenv(env)); v != "" && v != f.want {
				out = append(out, "the environment variable "+env)
			}
			key := string(role) + "." + f.field
			v, err := gitConfig(root, key)
			if err != nil {
				return nil, err
			}
			if v != "" && v != f.want {
				out = append(out, "the git config key "+key)
			}
		}
	}
	return out, nil
}

// WriteLocal sets user.name and user.email in the repository's own .git/config
// (never the global or system file), as the person running it: a plain git
// invocation, never a privileged one. Both fields are required.
func WriteLocal(root string, id Effective) error {
	if strings.TrimSpace(id.Name) == "" || strings.TrimSpace(id.Email) == "" {
		return fmt.Errorf("a git identity needs both a name and an email")
	}
	for _, kv := range [][2]string{{"user.name", id.Name}, {"user.email", id.Email}} {
		cmd := exec.Command("git", "-C", root, "config", "--local", kv[0], kv[1])
		cmd.Env = gitutil.ScrubbedEnv()
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git config --local %s: %w: %s", kv[0], err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// gitConfigScope is gitConfig restricted to one config file scope (--global,
// --local), with the same unset-is-empty reading.
func gitConfigScope(root, scope, key string) (string, error) {
	cmd := exec.Command("git", "-C", root, "config", scope, "--get", key)
	cmd.Env = gitutil.ScrubbedEnv()
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("git config %s %s: %w", scope, key, err)
	}
	return strings.TrimSpace(string(out)), nil
}
