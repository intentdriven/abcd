package runner

import (
	"testing"

	"github.com/intentdriven/abcd/internal/core/needs"
)

// TestEveryRunnersCommandIsReservedFromTheToolList: a repository's tool list
// cannot name the command a runner starts, so listing a tool can never move a
// registered copy of a harness ahead of the one the person installed
// (adr-2611041815445203, rule 2). needs cannot import this package, so this is
// where a runner added later is held to the reservation.
func TestEveryRunnersCommandIsReservedFromTheToolList(t *testing.T) {
	for _, name := range runnerNames {
		if !needs.Reserved(name) {
			t.Errorf("the runner %s starts %q, which the tool list does not reserve", name, name)
		}
	}
}
