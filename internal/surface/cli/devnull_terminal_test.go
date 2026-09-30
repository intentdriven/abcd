package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestConsentGatesTreatTheNullDeviceAsNoTerminal pins that stdin redirected
// from /dev/null — the shape a closed fd 0 takes too, since the Go runtime
// reopens it there — is no person at a terminal. /dev/null is a character
// device, so a device-mode check asked the install question, read EOF, and
// declined as "answered no at the terminal"; the gate must instead decline
// before asking, as having no terminal to ask at, and the itd-131 identity
// offer (which reads AtTerminal) must not ask either.
func TestConsentGatesTreatTheNullDeviceAsNoTerminal(t *testing.T) {
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	cmd := &cobra.Command{}
	cmd.SetIn(null)
	var errOut bytes.Buffer
	cmd.SetErr(&errOut)

	p := newPrompter(cmd)
	sp, ok := p.(*stdinPrompter)
	if !ok {
		t.Fatalf("newPrompter returned %T, want the stdin prompter", p)
	}
	if sp.AtTerminal() {
		t.Fatal("stdin from /dev/null reads as a person at a terminal")
	}
	ans := toolConfirm(p, nil, false, &errOut)(gitleaksExplained())
	if ans.Yes {
		t.Fatal("stdin from /dev/null installed a tool")
	}
	if !strings.HasPrefix(ans.Why, "no terminal to ask at") {
		t.Errorf("the decline reads %q, want it to say there is no terminal to ask at", ans.Why)
	}
	if strings.Contains(errOut.String(), "[y/N]") {
		t.Errorf("the install question was asked with no one to answer it:\n%s", errOut.String())
	}
}

// TestReadKeyTakesTheNullDeviceAsAnEmptyPipe pins the provider-key reader's
// sibling of the same check: stdin from /dev/null is not a terminal the key
// would be echoed on, so the refusal is that no key arrived.
func TestReadKeyTakesTheNullDeviceAsAnEmptyPipe(t *testing.T) {
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	_, err = readKey(null)
	if err == nil {
		t.Fatal("readKey accepted an empty /dev/null as a key")
	}
	if strings.Contains(err.Error(), "stdin is a terminal") {
		t.Errorf("readKey called /dev/null a terminal: %v", err)
	}
	if !strings.Contains(err.Error(), "none arrived on stdin") {
		t.Errorf("readKey's refusal does not say no key arrived: %v", err)
	}
}
