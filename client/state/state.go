package state

type State int

const (
	INIT_REBOOT State = iota
	REBOOTING
	INIT
	SELECTING
	REQUESTING
	BOUND
	RENEWING
	REBINDING
)

func (state State) String() string {
	if state < 0 || int(state) >= len(StringCandidates()) {
		return "UNRECOGNIZED_STATE"
	}

	return StringCandidates()[state]
}

func Parse(str string) State {
	for i, val := range StringCandidates() {
		if val == str {
			return State(i)
		}
	}

	return INIT
}

func StringCandidates() []string {
	return []string{"INIT_REBOOT", "REBOOTING", "INIT", "SELECTING", "REQUESTING", "BOUND", "RENEWING",
		"REBINDING"}
}
