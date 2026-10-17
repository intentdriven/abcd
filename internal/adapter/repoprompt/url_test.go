package repoprompt

import "testing"

// TestTheAddressIsThisMachine: a review request (the diff and the record) is
// sent only to this machine, so the MCP server's URL is admitted only as http
// or https to localhost or a loopback address, with no credentials, query or
// fragment.
func TestTheAddressIsThisMachine(t *testing.T) {
	for _, ok := range []string{"http://127.0.0.1:7777/mcp", "http://localhost:7777", "http://LOCALHOST:7777/mcp",
		"https://[::1]:7777/mcp", "http://127.1.2.3/"} {
		if err := CheckURL(ok); err != nil {
			t.Fatalf("%s is this machine: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "http://example.com/mcp", "https://10.0.0.1/mcp", "ftp://127.0.0.1/",
		"http://u:p@127.0.0.1/", "http://127.0.0.1/?q=1", "http://127.0.0.1/#f", "127.0.0.1:7777", "http:///mcp",
		"http://localhost.example.com/"} {
		if err := CheckURL(bad); err == nil {
			t.Fatalf("%q is refused", bad)
		}
	}
}

// TestARefusalNeverQuotesTheURL: a URL can carry a credential, so no refusal
// echoes it.
func TestARefusalNeverQuotesTheURL(t *testing.T) {
	for _, bad := range []string{"http://user:s3cret@127.0.0.1/", "https://s3cret.example.com/mcp"} {
		err := CheckURL(bad)
		if err == nil {
			t.Fatalf("%q is refused", bad)
		}
		if got := err.Error(); len(got) == 0 || contains(got, "s3cret") {
			t.Fatalf("the refusal quotes the URL: %q", got)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
