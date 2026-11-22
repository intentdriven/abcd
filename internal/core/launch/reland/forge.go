package reland

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// DefaultForge is GitHub through abcd's one door to its API
// (internal/core/github): the caller's own `gh`, on the repository origin
// names.
func DefaultForge() Forge {
	return Forge{ReadPR: readPR, Pusher: pusher, ListOpen: listOpen}
}

func (f Forge) withDefaults() Forge {
	d := DefaultForge()
	if f.ReadPR == nil {
		f.ReadPR = d.ReadPR
	}
	if f.Pusher == nil {
		f.Pusher = d.Pusher
	}
	if f.ListOpen == nil {
		f.ListOpen = d.ListOpen
	}
	return f
}

// originRepo is the owner/name origin's URL names on github.com.
func originRepo(root string) (string, string, error) {
	url, err := gitutil.RunCapped(root, 4096, "remote", "get-url", "origin")
	if err != nil || strings.TrimSpace(url) == "" {
		return "", "", errors.New("this checkout has no origin remote to ask about")
	}
	return github.ParseRemote(strings.TrimSpace(url))
}

// restPR is the pull-request endpoint's answer, the fields the workflow's
// event carries.
type restPR struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	User   struct {
		Login string `json:"login"`
	} `json:"user"`
	Head struct {
		Ref  string `json:"ref"`
		SHA  string `json:"sha"`
		Repo *struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"head"`
	Base struct {
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"base"`
}

func (p restPR) pullRequest() PullRequest {
	out := PullRequest{Number: p.Number, Title: p.Title, State: p.State, Author: p.User.Login,
		HeadRef: p.Head.Ref, HeadSHA: p.Head.SHA, Repo: p.Base.Repo.FullName}
	if p.Head.Repo != nil {
		out.HeadRepo = p.Head.Repo.FullName
	}
	return out
}

func readPR(root string, n int) (PullRequest, error) {
	owner, name, err := originRepo(root)
	if err != nil {
		return PullRequest{}, err
	}
	out, err := github.API(root, nil, "repos/"+owner+"/"+name+"/pulls/"+strconv.Itoa(n))
	if err != nil {
		return PullRequest{}, err
	}
	var p restPR
	if err := json.Unmarshal(out, &p); err != nil || p.Number != n {
		return PullRequest{}, fmt.Errorf("the forge's answer for #%d did not parse as that pull request", n)
	}
	return p.pullRequest(), nil
}

// listPage is how many open pull requests one listing reads.
const listPage = 100

func listOpen(root string) ([]PullRequest, error) {
	owner, name, err := originRepo(root)
	if err != nil {
		return nil, err
	}
	out, err := github.API(root, nil, "repos/"+owner+"/"+name+"/pulls?state=open&per_page="+strconv.Itoa(listPage))
	if err != nil {
		return nil, err
	}
	var ps []restPR
	if err := json.Unmarshal(out, &ps); err != nil {
		return nil, fmt.Errorf("the forge's listing of open pull requests did not parse")
	}
	prs := make([]PullRequest, 0, len(ps))
	for _, p := range ps {
		prs = append(prs, p.pullRequest())
	}
	return prs, nil
}

// pusher reads the forge's activity record of ref for the push that moved it
// to sha, newest first, and names its actor.
func pusher(ctx context.Context, root, ref, sha string) (string, error) {
	page, err := github.PushersPage(ctx, root, ref, "")
	if err != nil {
		return "", err
	}
	for _, p := range page.Pushes {
		if p.After == sha {
			return p.Login, nil
		}
	}
	return "", fmt.Errorf("no push to %s in the newest page of its activity record moved it to that commit", ref)
}

// Waiting is the listing: the declaration and the open pull requests that
// fall under one of its rows, each a bump a person relands while the
// workflow cannot.
type Waiting struct {
	Declaration Declaration   `json:"declaration"`
	Bumps       []PullRequest `json:"bumps"`
	// Truncated is a listing that read a full page, so more may be open.
	Truncated bool `json:"truncated,omitempty"`
}

// List asks the forge for the open pull requests a declared bot opened from
// a declared prefix in this repository: the bumps waiting on a reland.
func List(root string, f Forge) (Waiting, error) {
	d, err := readDeclaration(root)
	if err != nil {
		return Waiting{}, err
	}
	prs, err := f.withDefaults().ListOpen(root)
	if err != nil {
		return Waiting{}, fmt.Errorf("listing the open pull requests: %w", err)
	}
	w := Waiting{Declaration: d, Bumps: []PullRequest{}, Truncated: len(prs) >= listPage}
	for _, p := range prs {
		if _, ok := d.Matches(p.Author, p.HeadRef); ok && p.HeadRepo == p.Repo {
			w.Bumps = append(w.Bumps, p)
		}
	}
	return w, nil
}
