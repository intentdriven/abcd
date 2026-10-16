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

// MaxPreflightSlots bounds the count. The product thinker approved a measured
// trial of two preflights at once (dcn-2610161814279204), so two is the most
// either place may set until a measured trial supports more; a larger value
// is refused saying so (slotsRangeError).
const MaxPreflightSlots = 2

// slotsRangeError is the refusal of a count outside one to MaxPreflightSlots.
func slotsRangeError(n int) error {
	if n > MaxPreflightSlots {
		return fmt.Errorf("at most %d preflights may be in their test stages at once for now; the limit rises once a measured trial supports it (want a whole number from 1 to %d)",
			MaxPreflightSlots, MaxPreflightSlots)
	}
	return fmt.Errorf("want a whole number from 1 to %d", MaxPreflightSlots)
}

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
	// GateRace is the race gate, before the race stage (--race-gate).
	GateRace = "race"
)

// PreflightSlots resolves how many preflights may be in their test stages at
// once, and where that came from, for the account whose home is home. It
// reads getenv and the machine's config.json, and no repository file.
func PreflightSlots(getenv func(string) string, home string) (int, string, error) {
	if v := strings.TrimSpace(getenv(PreflightSlotsEnv)); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			n = 0
		}
		if n < 1 || n > MaxPreflightSlots {
			return 0, "", fmt.Errorf("%s is %q: %w", PreflightSlotsEnv, layered.BoundKey(v), slotsRangeError(n))
		}
		return n, SlotsEnv, nil
	}
	stack, err := layered.Load(layered.Config, layered.Roots{Home: home})
	if err != nil {
		return 0, "", machineConfigError(err)
	}
	if err := stack.Claim("load", "preflights"); err != nil {
		return 0, "", machineConfigError(err)
	}
	v, err := layered.Get(stack, PreflightSlotsKey, 1, func(n int) error {
		if n < 1 || n > MaxPreflightSlots {
			return slotsRangeError(n)
		}
		return nil
	})
	if err != nil {
		return 0, "", machineConfigError(err)
	}
	if v.Layer == layered.Machine {
		return v.V, SlotsMachine, nil
	}
	return v.V, SlotsDefault, nil
}

// machineConfigError frames a machine config.json that cannot be read for the
// count. It still refuses, as every layered read does: a file read past would
// drop whatever it sets without a word. But this refusal stops the first gate,
// and so every make preflight of the account, including a default user's who
// never set the count, so it names the file and says how to fix it.
func machineConfigError(err error) error {
	file := layered.Config.MachineOrigin()
	return fmt.Errorf("the machine configuration %s cannot be read for %s, so the load gate stops rather than guess how many preflights may be in their test stages at once: %w. "+
		"To fix it, keep %s a regular file you own that nobody else can write (chmod go-w %s), holding one valid JSON object whose \"load\" names no key but \"preflights\", a whole number from 1 to %d; "+
		"or move it aside, which drops every machine setting it holds, and one preflight at a time applies",
		file, PreflightSlotsKey, err, file, file, MaxPreflightSlots)
}
