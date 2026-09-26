package site

import (
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
	"github.com/intentdriven/abcd/internal/adapter/hosting/cloudflare"
)

// adapters is the provider list: every hosting adapter abcd ships, by manifest
// key. A second provider is one implementation of hosting.Adapter and one entry
// here; the verb does not change (itd-2609061543533170 criterion 3).
var adapters = []hosting.Adapter{cloudflare.Adapter{}}

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
