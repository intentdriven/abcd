package site

import (
	"context"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
	"github.com/intentdriven/abcd/internal/adapter/hosting/cloudflare"
	"github.com/intentdriven/abcd/internal/adapter/hosting/githubpages"
	"github.com/intentdriven/abcd/internal/core/ahoy"
)

// adapters is the provider list: every hosting adapter abcd ships, by manifest
// key. A second provider is one implementation of hosting.Adapter and one entry
// here; the verb does not change (itd-2609061543533170 criterion 3).
var adapters = []hosting.Adapter{cloudflare.Adapter{}, githubpages.Adapter{API: ghPagesAPI(ahoy.GH)}}

// ghPagesAPI is the github-pages provider's forge: `gh api` as the person who
// invoked the verb, exactly as the forge stage speaks (ghForge.api), so abcd
// holds no GitHub token for it either.
func ghPagesAPI(gh func(dir string, stdin []byte, args ...string) ([]byte, error)) githubpages.API {
	return func(_ context.Context, method, path string, body []byte) ([]byte, error) {
		args := []string{"api", "--hostname", ahoy.GitHubHost, "-H", "Accept: application/vnd.github+json"}
		if method != "" && method != "GET" {
			args = append(args, "--method", method)
		}
		args = append(args, path)
		if body != nil {
			args = append(args, "--input", "-")
		}
		return gh("", body, args...)
	}
}

// DefaultProvider is the provider setup routes through when the manifest names
// none.
const DefaultProvider = "cloudflare"

// Providers is the adapter list, by name, in the order abcd ships them.
func Providers() []string {
	out := make([]string, 0, len(adapters))
	for _, a := range adapters {
		out = append(out, a.Name())
	}
	return out
}

func adapterNamed(name string) (hosting.Adapter, bool) {
	for _, a := range adapters {
		if a.Name() == name {
			return a, true
		}
	}
	return nil, false
}

func providerList() string { return strings.Join(Providers(), ", ") }
