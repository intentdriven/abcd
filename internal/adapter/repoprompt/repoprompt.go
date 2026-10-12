// Package repoprompt is the RepoPrompt reviewer route's adapter (itd-6,
// spc-2609211950427074): it reaches the MCP server the person configured for
// RepoPrompt on their own machine, and RepoPrompt answers with whichever model
// the person configured in it.
//
// The adapter connects and nothing else. It never starts RepoPrompt or any
// process, never reads or writes RepoPrompt's configuration, and never looks
// for a server: the one address it reaches is the URL its caller hands it,
// which the person set (oracle.rp.url in the machine's configuration). The
// address is admitted only on this machine (CheckURL), since what a review
// request carries is the lane's diff and its record.
package repoprompt

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

// DefaultTool is the RepoPrompt tool a review request is sent to when the
// configuration names none (oracle.rp.tool): the one that sends a message to
// the model the person configured in RepoPrompt.
const DefaultTool = "oracle_send"

// CheckURL admits the MCP server's URL: an absolute http or https URL to
// localhost or a loopback address, with no credentials, query or fragment. A
// refusal never quotes the URL, which may carry a credential.
func CheckURL(raw string) error {
	if raw == "" {
		return errors.New("the URL is empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("it is not a URL")
	}
	switch {
	case u.Scheme != "http" && u.Scheme != "https":
		return errors.New("it is not an http or https URL")
	case u.User != nil:
		return errors.New("it carries credentials, and the MCP server's address takes none")
	case u.RawQuery != "" || u.ForceQuery:
		return errors.New("it carries a query, and the MCP server's address takes none")
	case u.Fragment != "" || strings.Contains(raw, "#"):
		return errors.New("it carries a fragment, and the MCP server's address takes none")
	case u.Hostname() == "":
		return errors.New("it names no host")
	case !loopback(u.Hostname()):
		return errors.New("it names another machine; a review request (the lane's diff and its record) is sent only to this machine, to localhost or a loopback address")
	}
	return nil
}

// loopback reports whether host is localhost or a loopback address.
func loopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
