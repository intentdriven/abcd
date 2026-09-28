package site

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/ahoy"
)

// Forge is the slice of the forge `abcd site setup` reads and writes: the
// deployment environments the site workflow runs in, and the NAMES of the
// secrets one of them holds. It never reads or writes a secret's value — the
// forge encrypts those, and setting one is the person's step.
type Forge interface {
	// Repo names the repository every call acts on.
	Repo() string
	// DefaultBranch reads the repository's default branch: the branch the
	// workflow gates on and the environments admit.
	DefaultBranch(ctx context.Context) (string, error)
	// Environments reads every environment the repository has, by name.
	Environments(ctx context.Context) (map[string]EnvironmentState, error)
	// Policies reads one environment's deployment branch and tag policies.
	Policies(ctx context.Context, env string) ([]BranchPolicy, error)
	// PutEnvironment creates env restricted to custom policies. The forge's
	// write replaces an environment's whole protection set, so setup calls it
	// only for an environment that does not exist.
	PutEnvironment(ctx context.Context, env string) error
	// AddPolicy admits one branch or tag pattern to env.
	AddPolicy(ctx context.Context, env string, p BranchPolicy) error
	// SecretNames lists the names of env's secrets. Values are never read.
	SecretNames(ctx context.Context, env string) ([]string, error)
}

// EnvironmentState is one environment's deployment policy as the forge reports
// it. An environment with no policy at all admits every ref.
type EnvironmentState struct {
	ProtectedBranches    bool
	CustomBranchPolicies bool
}

// BranchPolicy is one deployment policy: a branch or tag name pattern.
type BranchPolicy struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ghForge is Forge over the GitHub CLI, acting as the person who invoked the
// verb (ahoy.GH): abcd holds no forge token.
type ghForge struct {
	dir  string
	repo string
	// gh runs one gh subcommand; nil is ahoy.GH. Tests point it at an
	// in-process fake of the forge's API, so nothing reaches a network.
	gh func(dir string, stdin []byte, args ...string) ([]byte, error)
}

// GitHubForge is the forge for the checkout at dir, or an error saying why the
// checkout names no GitHub repository abcd may act on.
func GitHubForge(dir string) (Forge, error) {
	repo, err := ahoy.GitHubRepo(dir)
	if err != nil {
		return nil, err
	}
	return &ghForge{dir: dir, repo: repo}, nil
}

func (g *ghForge) Repo() string { return g.repo }

func (g *ghForge) api(stdin []byte, method, path string) ([]byte, error) {
	args := []string{"api", "--hostname", ahoy.GitHubHost, "-H", "Accept: application/vnd.github+json"}
	if method != "" {
		args = append(args, "--method", method)
	}
	args = append(args, path)
	if stdin != nil {
		args = append(args, "--input", "-")
	}
	if g.gh != nil {
		return g.gh(g.dir, stdin, args...)
	}
	return ahoy.GH(g.dir, stdin, args...)
}

// envPath builds an environment path. The environment names are this
// package's own constants, escaped regardless.
func (g *ghForge) envPath(env, rest string) string {
	return "repos/" + g.repo + "/environments/" + url.PathEscape(env) + rest
}

func (g *ghForge) DefaultBranch(context.Context) (string, error) {
	out, err := g.api(nil, "", "repos/"+g.repo)
	if err != nil {
		return "", err
	}
	var doc struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return "", errors.New("the repository response could not be read as JSON")
	}
	if doc.DefaultBranch == "" {
		return "", errors.New("the repository response names no default branch")
	}
	return doc.DefaultBranch, nil
}

// perPage is the largest page the forge serves a list read in.
const perPage = 100

// maxListPages bounds a list read: a forge that kept serving full pages past
// ten thousand entries is refused rather than followed.
const maxListPages = 100

// listAll reads every page of the list endpoint path, whose entries arrive
// under key, and hands each entry to each. The forge serves a list read at most
// perPage entries at a time and reports the whole list's total_count, so a read
// stops when it holds that many and fails when it cannot: a page that cannot be
// read, or that comes back short of the total, is an error rather than a
// shorter list. The first page's total is held for the whole read, and a later
// page reporting another is an error too: a list that changed between two page
// reads has shifted entries across the page boundary, so one was never seen
// even when the count served matches the later total (iss-2609261241117925).
// Every list setup reads decides whether it writes (an environment read as
// absent is created through the endpoint that replaces its whole protection
// set), so a partial list is never an answer.
func (g *ghForge) listAll(path, key, what string, each func(json.RawMessage) error) error {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	got, first := 0, -1
	for page := 1; ; page++ {
		if page > maxListPages {
			return fmt.Errorf("the %s list runs past %d pages, so it was not read", what, maxListPages)
		}
		out, err := g.api(nil, "", path+sep+"per_page="+strconv.Itoa(perPage)+"&page="+strconv.Itoa(page))
		if err != nil {
			return err
		}
		var doc map[string]json.RawMessage
		var entries []json.RawMessage
		var total *int
		if json.Unmarshal(out, &doc) != nil || json.Unmarshal(doc["total_count"], &total) != nil || total == nil {
			return fmt.Errorf("the %s response could not be read as JSON", what)
		}
		if first < 0 {
			first = *total
		} else if *total != first {
			return fmt.Errorf("the forge reported %d %s on the first page and %d on page %d, so the list changed while it was read and was not read whole", first, what, *total, page)
		}
		// An empty list may arrive with no entries field at all.
		if raw, ok := doc[key]; (ok || *total > 0) && json.Unmarshal(raw, &entries) != nil {
			return fmt.Errorf("the %s response could not be read as JSON", what)
		}
		for _, e := range entries {
			if err := each(e); err != nil {
				return err
			}
		}
		got += len(entries)
		if got >= *total {
			if got > *total {
				return fmt.Errorf("the forge reported %d %s and served %d, so the list was not read whole", *total, what, got)
			}
			return nil
		}
		if len(entries) < perPage {
			return fmt.Errorf("the forge reported %d %s and served %d, so the list was not read whole", *total, what, got)
		}
	}
}

func (g *ghForge) Environments(context.Context) (map[string]EnvironmentState, error) {
	envs := map[string]EnvironmentState{}
	err := g.listAll("repos/"+g.repo+"/environments", "environments", "environments", func(raw json.RawMessage) error {
		var e struct {
			Name   string `json:"name"`
			Policy *struct {
				Protected bool `json:"protected_branches"`
				Custom    bool `json:"custom_branch_policies"`
			} `json:"deployment_branch_policy"`
		}
		if err := json.Unmarshal(raw, &e); err != nil {
			return errors.New("the environments response could not be read as JSON")
		}
		st := EnvironmentState{}
		if e.Policy != nil {
			st.ProtectedBranches, st.CustomBranchPolicies = e.Policy.Protected, e.Policy.Custom
		}
		envs[e.Name] = st
		return nil
	})
	if err != nil {
		return nil, err
	}
	return envs, nil
}

func (g *ghForge) Policies(_ context.Context, env string) ([]BranchPolicy, error) {
	var policies []BranchPolicy
	err := g.listAll(g.envPath(env, "/deployment-branch-policies"), "branch_policies", "deployment policies", func(raw json.RawMessage) error {
		var p BranchPolicy
		if err := json.Unmarshal(raw, &p); err != nil {
			return errors.New("the deployment policies response could not be read as JSON")
		}
		if p.Type == "" {
			p.Type = "branch"
		}
		policies = append(policies, p)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return policies, nil
}

func (g *ghForge) PutEnvironment(_ context.Context, env string) error {
	body, _ := json.Marshal(map[string]any{
		"deployment_branch_policy": map[string]bool{"protected_branches": false, "custom_branch_policies": true},
	})
	_, err := g.api(body, "PUT", g.envPath(env, ""))
	return err
}

func (g *ghForge) AddPolicy(_ context.Context, env string, p BranchPolicy) error {
	body, _ := json.Marshal(p)
	_, err := g.api(body, "POST", g.envPath(env, "/deployment-branch-policies"))
	return err
}

func (g *ghForge) SecretNames(_ context.Context, env string) ([]string, error) {
	names := []string{}
	err := g.listAll(g.envPath(env, "/secrets"), "secrets", "secrets", func(raw json.RawMessage) error {
		var s struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &s); err != nil {
			return errors.New("the secrets response could not be read as JSON")
		}
		names = append(names, s.Name)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return names, nil
}
