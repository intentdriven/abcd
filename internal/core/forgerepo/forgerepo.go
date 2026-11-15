// Package forgerepo says whether two repository addresses name one
// repository, and whether a remote's every address names the repository the
// forge client resolves. It is the one comparison abcd makes of repository
// URLs: `abcd land` (internal/core/landing) holds its push and its pull
// request to it, and the sweep (internal/core/sweep) holds its deletes on
// origin and its forge calls to it.
//
// A key is a URL's comparable form, so an https and an ssh address of one
// repository, or two spellings that differ in a user, a default port, a
// trailing slash, a .git suffix or case on a host, compare equal. It reads
// only the text it is given: it runs nothing and resolves no host.
package forgerepo

import (
	"net/url"
	"path/filepath"
	"strings"
)

// defaultPorts are the ports a URL scheme takes when it names none.
var defaultPorts = map[string]string{"ssh": "22", "git+ssh": "22", "https": "443", "http": "80", "git": "9418"}

// Key is a repository URL's comparable form, "host[:port]:/path" with the
// host and (on a host) the path lower-cased, any user, trailing slash and .git
// suffix dropped, and a port kept unless it is the scheme's own; a local path
// or file URL has an empty host. "" when the URL cannot be read.
func Key(u string) string {
	u = strings.TrimSpace(u)
	var host, path string
	switch c := strings.Index(u, ":"); {
	case u == "":
		return ""
	case strings.Contains(u, "://"):
		p, err := url.Parse(u)
		if err != nil {
			return ""
		}
		host, path = strings.ToLower(p.Hostname()), p.Path
		if port := p.Port(); port != "" && port != defaultPorts[strings.ToLower(p.Scheme)] {
			host += ":" + port
		}
		if p.Scheme == "file" {
			host = ""
		}
	case c > 1 && !strings.Contains(u[:c], "/"):
		h := u[:c]
		if at := strings.LastIndex(h, "@"); at >= 0 {
			h = h[at+1:]
		}
		host, path = strings.ToLower(h), "/"+strings.TrimPrefix(u[c+1:], "/")
	default:
		path = u
	}
	if host == "" {
		path = filepath.Clean(path)
	}
	path = strings.TrimSuffix(strings.TrimSuffix(path, "/"), ".git")
	path = strings.TrimSuffix(path, "/")
	if host != "" {
		path = strings.ToLower(path)
	}
	if path == "" || path == "/" || path == "." {
		return ""
	}
	return host + ":" + path
}

// SplitKey splits a Key into its host (with any port) and its path.
func SplitKey(k string) (host, path string) {
	if p, ok := strings.CutPrefix(k, ":"); ok {
		return "", p
	}
	if i := strings.Index(k, ":/"); i >= 0 {
		return k[:i], k[i+1:]
	}
	return "", k
}

// Keys is the set of keys of the forge's own addresses for one repository
// (`gh repo view --json url,sshUrl`), leaving out any that cannot be read.
func Keys(urls ...string) map[string]bool {
	keys := map[string]bool{}
	for _, u := range urls {
		if k := Key(u); k != "" {
			keys[k] = true
		}
	}
	return keys
}

// Stranger is the first of a remote's addresses that does not name a
// repository in keys, with its key ("" when it cannot be read), and whether
// there is one. A remote none of whose addresses is a stranger is the
// forge's repository at every address it is fetched from or pushed to.
func Stranger(urls []string, keys map[string]bool) (u, key string, found bool) {
	for _, u := range urls {
		k := Key(u)
		if k != "" && keys[k] {
			continue
		}
		return u, k, true
	}
	return "", "", false
}
