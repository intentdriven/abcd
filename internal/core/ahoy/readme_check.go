package ahoy

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The three checks every derived README body passes before it is written
// (spc-2610031704338417 step 2), in order:
//
//  1. the badge-address check (decision 7; R6), on a badge part's addresses;
//  2. the outbound scrub, scanner.ScrubOutbound, on every body;
//  3. the project's own docs-lint banned tokens, where it keeps
//     .abcd/docs-lint.json.
//
// A part failing any one is omitted with the check's finding named; nothing
// here writes. The record makes no claim that scanner.OutboundPolicy covers
// badge addresses: the first check is what covers them.

// readmeBadgeHosts is the one allowed list the badge-address check holds: per
// badge part, the host its link may name and the host its image may name.
var readmeBadgeHosts = map[string]map[string]string{
	"release-badge": {readmeAddrLink: readmeForgeHost, readmeAddrImage: readmeBadgeImageHost},
	"build-badge":   {readmeAddrLink: readmeForgeHost, readmeAddrImage: readmeForgeHost},
	"abcd-badge":    {readmeAddrLink: readmeForgeHost, readmeAddrImage: readmeBadgeImageHost},
}

// readmeBadgeQueryKeys are the only query keys an address may carry: the ones
// the forge reads on its workflow status badge. Every other address carries
// none.
var readmeBadgeQueryKeys = map[string]map[string]bool{
	"build-badge/" + readmeAddrImage: {"branch": true, "event": true},
}

// plainQueryKeyRe is a query key a refusal may repeat as it stands.
var plainQueryKeyRe = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)

// checkBadgeAddress is the badge-address check. The address is parsed with
// net/url, used only as a parser. The scheme must be https; no credentials and
// no fragment; the host the one readmeBadgeHosts allows for the part and the
// role; and a query only where readmeBadgeQueryKeys allows it, with only its
// keys. A refusal leads with "<part> refused: " and names the piece: the
// scheme, the host or the query key, never a query value.
func checkBadgeAddress(part string, a readmeAddr) error {
	refuse := func(format string, args ...any) error {
		return fmt.Errorf("%s refused: "+format, append([]any{part}, args...)...)
	}
	hosts, ok := readmeBadgeHosts[part]
	if !ok {
		return refuse("not a badge part, so it carries no address")
	}
	host, ok := hosts[a.Role]
	if !ok {
		return refuse("a badge carries only a link and an image")
	}
	if strings.IndexFunc(a.URL, func(r rune) bool { return r <= ' ' || r == 0x7f }) >= 0 {
		return refuse("the %s is not an address abcd can read", a.Role)
	}
	u, err := url.Parse(a.URL)
	if err != nil {
		return refuse("the %s is not an address abcd can read", a.Role)
	}
	if u.Scheme != "https" {
		scheme := u.Scheme
		if scheme == "" {
			scheme = "(none)"
		}
		return refuse("the %s address's scheme is not https: %s", a.Role, scheme)
	}
	if u.Opaque != "" || u.Host == "" {
		return refuse("the %s is not an address abcd can read", a.Role)
	}
	if u.User != nil {
		return refuse("the %s address carries credentials", a.Role)
	}
	if u.Host != host {
		return refuse("the %s address's host is not %s: %s", a.Role, host, termsafe.Sanitize(u.Host))
	}
	if u.RawQuery != "" {
		q, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return refuse("the %s address's query string cannot be read", a.Role)
		}
		keys := make([]string, 0, len(q))
		for k := range q {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		allowed := readmeBadgeQueryKeys[part+"/"+a.Role]
		for _, k := range keys {
			if allowed[k] {
				continue
			}
			if !plainQueryKeyRe.MatchString(k) {
				return refuse("the address carries a query key the forge does not use, and not one abcd will repeat")
			}
			return refuse("the address carries a query key the forge does not use: %s", k)
		}
	}
	if u.Fragment != "" || u.RawFragment != "" {
		return refuse("the %s address carries a fragment", a.Role)
	}
	return nil
}

// readmeChecks holds what the checks load once per derivation: the project's
// docs-lint bans, or why they did not load.
type readmeChecks struct {
	root    string
	lint    *lint.TokenChecker
	lintWhy string
}

// newReadmeChecks loads the project's docs-lint bans. A project with no
// .abcd/docs-lint.json skips the third check; one whose file does not load or
// whose bans do not compile refuses every part with the reason, as a scanner
// configuration that does not load does, since a gate that silently stops
// judging is a gate passing by not looking.
func newReadmeChecks(root string) *readmeChecks {
	c := &readmeChecks{root: root}
	rt, err := os.OpenRoot(root)
	if err != nil {
		c.lintWhy = "the project's .abcd/docs-lint.json could not be read: the project root cannot be opened"
		return c
	}
	defer rt.Close()
	cfg, err := lint.LoadConfigInRoot(rt, ".abcd/docs-lint.json")
	switch {
	case os.IsNotExist(err):
		return c
	case err != nil:
		c.lintWhy = "the project's .abcd/docs-lint.json does not load: " + termsafe.Sanitize(err.Error())
		return c
	}
	tc, err := lint.NewTokenChecker(cfg.BannedTokens)
	if err != nil {
		c.lintWhy = "the project's .abcd/docs-lint.json bans do not compile: " + termsafe.Sanitize(err.Error())
		return c
	}
	c.lint = tc
	return c
}

// run passes one derived body through the three checks in order. It returns
// the body when every check passes, else the first check's finding as the
// reason, with any warning noted.
func (c *readmeChecks) run(name, body string, addrs []readmeAddr) (string, string, []string) {
	refuse := func(why string) (string, string, []string) { return "", name + " refused: " + why, nil }

	if _, badge := readmeBadgeHosts[name]; badge {
		if len(addrs) == 0 {
			return refuse("a badge with no address to check")
		}
		for _, a := range addrs {
			if err := checkBadgeAddress(name, a); err != nil {
				return "", err.Error(), nil
			}
		}
	}

	_, findings, err := scanner.ScrubOutbound(c.root, body, "README.md part "+name)
	if err != nil {
		return refuse("the outbound scrub refused it: " + termsafe.Sanitize(err.Error()))
	}
	if len(findings) > 0 {
		// The rule, never the matched text: a part is never written with a
		// masked span inside it, and the summary does not repeat what it masked.
		kinds := map[string]bool{}
		for _, f := range findings {
			kinds[f.Kind] = true
		}
		return refuse("the outbound scrub found " + strings.Join(readmeSortedKeys(kinds), ", "))
	}

	if c.lintWhy != "" {
		return refuse(c.lintWhy)
	}
	var notes []string
	if c.lint != nil {
		blockers := map[string]bool{}
		for _, f := range c.lint.LintComposed("README.md", body, false, nil) {
			if f.Severity == "blocker" {
				blockers[f.RuleID] = true
				continue
			}
			notes = append(notes, fmt.Sprintf("%s part: docs-lint %s %s at its line %d; written", name, f.Severity, f.RuleID, f.Line))
		}
		if len(blockers) > 0 {
			return refuse("the project's docs-lint bans " + strings.Join(readmeSortedKeys(blockers), ", "))
		}
	}
	return body, "", notes
}

func readmeSortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
