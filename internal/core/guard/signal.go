package guard

import "strings"

// signalWordCommands names the commands whose first `-NAME` word naming a
// signal is the signal to send, whatever the case of its letters, and not a
// cluster of short options. procps-ng pkill reads the first such word anywhere
// in its arguments, BSD pkill reads it as the first argument, and both compare
// the name case-insensitively, with or without its SIG prefix. So `pkill -term
// -g 4242` sends TERM to a group, and the `t` in it is no terminal selector;
// read as a cluster, the letters of `-HUP`, `-USR1` and `-SEGV` would be the
// user and group selectors `-U` and `-G`. psmisc killall is not here: it reads
// a signal name only when it begins with a capital letter, and hands a
// lower-case one to its option parser, so `killall -term` is `killall -t erm`.
var signalWordCommands = map[string]bool{"pkill": true}

// signalNames are the signal names the pkill implementations accept, Linux
// and BSD together, without the SIG prefix. A name one platform lacks is
// refused as an option there, so reading it as a signal spares nothing that
// runs. A name no pkill accepts is not here: its letters are options, so
// `pkill -null` is `-n -u ll` and BSD's `pkill -unused` is `-u nused`.
var signalNames = map[string]bool{
	"hup": true, "int": true, "quit": true, "ill": true, "trap": true, "abrt": true,
	"iot": true, "bus": true, "emt": true, "fpe": true, "kill": true, "usr1": true,
	"segv": true, "usr2": true, "pipe": true, "alrm": true, "term": true, "stkflt": true,
	"chld": true, "cld": true, "cont": true, "stop": true, "tstp": true, "ttin": true,
	"ttou": true, "urg": true, "xcpu": true, "xfsz": true, "vtalrm": true, "prof": true,
	"winch": true, "io": true, "poll": true, "pwr": true, "sys": true, "info": true,
	"lost": true, "rtmin": true, "rtmax": true,
}

// isSignalWord reports whether a word is `-` and a signal: a number, or a name
// in any case, with or without the SIG prefix, and a real-time signal offset
// from its bound (`-RTMIN+3`).
func isSignalWord(tok string) bool {
	if len(tok) < 2 || tok[0] != '-' {
		return false
	}
	name := strings.ToLower(tok[1:])
	if allDigits(name) {
		return true
	}
	name = strings.TrimPrefix(name, "sig")
	if signalNames[name] {
		return true
	}
	for _, bound := range []string{"rtmin", "rtmax"} {
		if rest, ok := strings.CutPrefix(name, bound); ok && len(rest) > 1 &&
			(rest[0] == '+' || rest[0] == '-') && allDigits(rest[1:]) {
			return true
		}
	}
	return false
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
