package site

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"

	"github.com/intentdriven/abcd/internal/core/ahoy"
)

// Forge is the slice of the forge `abcd site setup` reads and writes: the
// deployment environments the site workflow runs in, and the NAMES of the
// secrets one of them holds. It never reads or writes a secret's value — the
// forge encrypts those, and setting one is the person's step.
type Forge interface {
	// Repo names the repository every call acts on.
	Repo() string
	// Environments reads every environment the repository has, by name.
	Environments(ctx context.Context) (map[string]EnvironmentState, error)
	// Policies reads one environment's deployment branch and tag policies.
	Policies(ctx context.Context, env string) ([]BranchPolicy, error)
	// PutEnvironment creates env, or updates it, restricted to custom policies.
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
	return ahoy.GH(g.dir, stdin, args...)
}

// envPath builds an environment path. The environment names are this
// package's own constants, escaped regardless.
func (g *ghForge) envPath(env, rest string) string {
	return "repos/" + g.repo + "/environments/" + url.PathEscape(env) + rest
}

func (g *ghForge) Environments(context.Context) (map[string]EnvironmentState, error) {
	out, err := g.api(nil, "", "repos/"+g.repo+"/environments?per_page=100")
	if err != nil {
		return nil, err
	}
	var doc struct {
		Environments []struct {
			Name   string `json:"name"`
			Policy *struct {
				Protected bool `json:"protected_branches"`
				Custom    bool `json:"custom_branch_policies"`
			} `json:"deployment_branch_policy"`
		} `json:"environments"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, errors.New("the environments response could not be read as JSON")
	}
	envs := map[string]EnvironmentState{}
	for _, e := range doc.Environments {
		st := EnvironmentState{}
		if e.Policy != nil {
			st.ProtectedBranches, st.CustomBranchPolicies = e.Policy.Protected, e.Policy.Custom
		}
		envs[e.Name] = st
	}
	return envs, nil
}

func (g *ghForge) Policies(_ context.Context, env string) ([]BranchPolicy, error) {
	out, err := g.api(nil, "", g.envPath(env, "/deployment-branch-policies?per_page=100"))
	if err != nil {
		return nil, err
	}
	var doc struct {
		Policies []BranchPolicy `json:"branch_policies"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, errors.New("the deployment policies response could not be read as JSON")
	}
	for i := range doc.Policies {
		if doc.Policies[i].Type == "" {
			doc.Policies[i].Type = "branch"
		}
	}
	return doc.Policies, nil
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
	out, err := g.api(nil, "", g.envPath(env, "/secrets?per_page=100"))
	if err != nil {
		return nil, err
	}
	var doc struct {
		Secrets []struct {
			Name string `json:"name"`
		} `json:"secrets"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, errors.New("the secrets response could not be read as JSON")
	}
	names := make([]string, 0, len(doc.Secrets))
	for _, s := range doc.Secrets {
		names = append(names, s.Name)
	}
	return names, nil
}
