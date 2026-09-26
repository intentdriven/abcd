package cli

import (
	"reflect"
	"strings"
	"testing"
)

// requirements_test.go — a verb's unmet requirements are named in ONE refusal
// (iss-2609100531051385). The positional-count refusal named no flag, and a
// caller missing a positional and a required flag paid one round trip per
// requirement. When more than one requirement is unmet, or the positionals are
// wrong, the refusal names every one and the verb's usage line.

func TestUsageRequirementsReadsTheUseLine(t *testing.T) {
	for _, tt := range []struct {
		use  string
		want [][]string
	}{
		{"resolve <iss-N> <note> --impact <additive|breaking|fix|internal> [--grounds \"<token>: <text>\"] [--intent itd-N]", [][]string{{"--impact"}}},
		{"add --private|--public <key> <pattern|->", [][]string{{"--private", "--public"}}},
		{"assemble --position <position> --target <HEAD|sha>", [][]string{{"--position"}, {"--target"}}},
		{"condition <itd-N> [<cond-id> --disposition <x> --occasioned-by <y> --grounds \"<why>\" [--narrowing \"<n>\"]]", nil},
		{"audit [<itd-N>] | audit --issue-drift [--strict]", nil},
		{"list [--private | --public]", nil},
		{"close <spc-N>", nil},
	} {
		if got := usageRequirements(tt.use); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("usageRequirements(%q) = %v, want %v", tt.use, got, tt.want)
		}
	}
}

func TestResolveNamesEveryUnmetRequirementAtOnce(t *testing.T) {
	captureLedgerRepo(t)
	_, err := runCLIErr(t, "capture", "resolve", "iss-1")
	if err == nil {
		t.Fatal("capture resolve with one positional and no --impact must refuse")
	}
	msg := err.Error()
	for _, want := range []string{"accepts 2 arg(s), received 1", "--impact", "usage: abcd capture resolve <iss-N> <note> --impact"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal does not name %q:\n%s", want, msg)
		}
	}
}

func TestAssembleNamesBothMissingFlagsAtOnce(t *testing.T) {
	captureLedgerRepo(t)
	_, err := runCLIErr(t, "reading", "assemble")
	if err == nil {
		t.Fatal("reading assemble with no flags must refuse")
	}
	for _, want := range []string{"--position", "--target"} {
		if !strings.Contains(err.Error(), want+" is not set") {
			t.Errorf("the refusal does not name %s as unset:\n%v", want, err)
		}
	}
}

// One unmet requirement keeps the verb's own refusal, which already names it
// with what the verb knows about it.
func TestASingleUnmetFlagKeepsTheVerbsOwnRefusal(t *testing.T) {
	captureLedgerRepo(t)
	_, err := runCLIErr(t, "capture", "resolve", "iss-1", "a note")
	if err == nil || !strings.Contains(err.Error(), "impact is required") {
		t.Fatalf("want the verb's own --impact refusal, got %v", err)
	}
}
