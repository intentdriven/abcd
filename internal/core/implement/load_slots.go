package implement

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/layered"
)

// How many preflights may be in their test stages at once (iss-2610161639025702)
// is a setting of the person's own machine, never of a checkout: a repository
// that could raise it could raise the machine's load for every account on it.
// So it is read from two places only, the first that holds it winning:
//
//   - ABCD_LOAD_PREFLIGHTS in the waiter's own environment, so a measured
//     trial can raise it without editing a file;
//   - load.preflights in ~/.abcd.noindex/config.json, the machine layer of the
//     layered configuration, loaded alone (layered.Roots carries no Repo), so
//     the guarded read refuses a file that is not a regular file this account
//     owns and nobody else can write, exactly as for every machine setting.
//
// Absent from both it is one, and every gate reads as it did before the
// setting existed. A value from either place that is not a whole number from
// one to MaxPreflightSlots is refused naming where it was set, never replaced
// by the default: a setting that silently did less than it says is the
// failure the layered reader exists to close. A later step derives the count
// from the machine's resources instead (iss-2610161756384596).

// PreflightSlotsEnv is the environment variable a trial sets the count with.
const PreflightSlotsEnv = "ABCD_LOAD_PREFLIGHTS"

// PreflightSlotsKey is the machine configuration key.
const PreflightSlotsKey = "load.preflights"

// MaxPreflightSlots bounds the count. At four, each of four concurrent test
// stages still takes a quarter of the cores, go test's -p floor
// (machineload.GoTestParallelism).
const MaxPreflightSlots = 4

// Where the count came from.
const (
	SlotsDefault = "default"
	SlotsMachine = "machine"
	SlotsEnv     = "env"
)

// The two gates a waiter waits at.
const (
	// GateTests is the first gate, before a preflight's test stages.
	GateTests = "tests"
	// GateRace is the race gate, before the race stage (--race).
	GateRace = "race"
)

// PreflightSlots resolves how many preflights may be in their test stages at
// once, and where that came from, for the account whose home is home. It
// reads getenv and the machine's config.json, and no repository file.
func PreflightSlots(getenv func(string) string, home string) (int, string, error) {
	if v := strings.TrimSpace(getenv(PreflightSlotsEnv)); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > MaxPreflightSlots {
			return 0, "", fmt.Errorf("%s is %q: want a whole number from 1 to %d", PreflightSlotsEnv, layered.BoundKey(v), MaxPreflightSlots)
		}
		return n, SlotsEnv, nil
	}
	stack, err := layered.Load(layered.Config, layered.Roots{Home: home})
	if err != nil {
		return 0, "", err
	}
	if err := stack.Claim("load", "preflights"); err != nil {
		return 0, "", err
	}
	v, err := layered.Get(stack, PreflightSlotsKey, 1, func(n int) error {
		if n < 1 || n > MaxPreflightSlots {
			return fmt.Errorf("want a whole number from 1 to %d", MaxPreflightSlots)
		}
		return nil
	})
	if err != nil {
		return 0, "", err
	}
	if v.Layer == layered.Machine {
		return v.V, SlotsMachine, nil
	}
	return v.V, SlotsDefault, nil
}
