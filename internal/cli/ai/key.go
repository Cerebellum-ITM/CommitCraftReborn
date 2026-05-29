package ai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"commit_craft_reborn/internal/config"
)

const keyUsage = `Usage: commitcraft ai key <action> [flags]

Actions:
  show              Print the current slot state as JSON (no secrets).
  set               Set a slot's key. Flags: --slot user|ai, --value <key>.
                    Missing flags are prompted for (the key value is read
                    without echo when stdin is a terminal).
  swap              Toggle the active slot user<->ai (errors if the target
                    slot has no key set).
  use               Set the active slot explicitly. Flag: --slot user|ai.

With no action, behaves like 'show'.
`

// keyStateJSON is the wire shape for `ai key show` / the post-mutation echo.
// It deliberately never carries a key value.
type keyStateJSON struct {
	ActiveSlot string `json:"active_slot"`
	UserKeySet bool   `json:"user_key_set"`
	AIKeySet   bool   `json:"ai_key_set"`
}

func toKeyStateJSON(s config.KeyState) keyStateJSON {
	return keyStateJSON{
		ActiveSlot: s.ActiveSlot,
		UserKeySet: s.UserKeySet,
		AIKeySet:   s.AIKeySet,
	}
}

func printKeyState(s config.KeyState) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(toKeyStateJSON(s))
}

// runKey dispatches the `commitcraft ai key` actions. Returns the process
// exit code (0 ok, 1 runtime error, 2 usage error).
func runKey(args []string) int {
	action := "show"
	rest := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		action, rest = args[0], args[1:]
	}

	switch action {
	case "show":
		return runKeyShow()
	case "set":
		return runKeySet(rest)
	case "swap":
		return runKeySwap()
	case "use":
		return runKeyUse(rest)
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, keyUsage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown key action %q\n\n%s", action, keyUsage)
		return 2
	}
}

func runKeyShow() int {
	state, err := config.LoadKeyState()
	if err != nil {
		printErrorJSON("config_error", err.Error())
		return 1
	}
	printKeyState(state)
	return 0
}

func runKeySet(args []string) int {
	fs := flagSet("ai key set")
	slot := fs.String("slot", "", "Slot to set: user|ai (prompted if omitted).")
	value := fs.String("value", "", "API key value (prompted without echo if omitted).")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	chosenSlot := strings.ToLower(strings.TrimSpace(*slot))
	if chosenSlot == "" {
		chosenSlot = promptSlot()
	}
	if chosenSlot != config.KeySlotUser && chosenSlot != config.KeySlotAI {
		printErrorJSON("invalid_input",
			fmt.Sprintf("--slot must be %q or %q (got %q)",
				config.KeySlotUser, config.KeySlotAI, chosenSlot))
		return 2
	}

	keyValue := strings.TrimSpace(*value)
	if keyValue == "" {
		entered, err := promptSecret(fmt.Sprintf("Groq API key for the %q slot: ", chosenSlot))
		if err != nil {
			printErrorJSON("input_error", err.Error())
			return 1
		}
		keyValue = strings.TrimSpace(entered)
	}
	if keyValue == "" {
		printErrorJSON("invalid_input",
			"empty key value — refusing to clear the slot (pass a non-empty key)")
		return 2
	}

	if err := config.SaveEnvVar(config.EnvVarForSlot(chosenSlot), keyValue); err != nil {
		printErrorJSON("config_error", err.Error())
		return 1
	}

	state, err := config.LoadKeyState()
	if err != nil {
		printErrorJSON("config_error", err.Error())
		return 1
	}
	printKeyState(state)
	return 0
}

func runKeySwap() int {
	state, err := config.LoadKeyState()
	if err != nil {
		printErrorJSON("config_error", err.Error())
		return 1
	}
	target := config.KeySlotAI
	if state.ActiveSlot == config.KeySlotAI {
		target = config.KeySlotUser
	}
	return activateSlot(state, target)
}

func runKeyUse(args []string) int {
	fs := flagSet("ai key use")
	slot := fs.String("slot", "", "Slot to activate: user|ai.")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	target := strings.ToLower(strings.TrimSpace(*slot))
	if target != config.KeySlotUser && target != config.KeySlotAI {
		printErrorJSON("invalid_input",
			fmt.Sprintf("--slot must be %q or %q (got %q)",
				config.KeySlotUser, config.KeySlotAI, target))
		return 2
	}
	state, err := config.LoadKeyState()
	if err != nil {
		printErrorJSON("config_error", err.Error())
		return 1
	}
	return activateSlot(state, target)
}

// activateSlot persists target as the active slot, refusing to switch into a
// slot that has no key stored so the user can't strand the pipeline on an
// empty slot.
func activateSlot(state config.KeyState, target string) int {
	if !state.SlotSet(target) {
		printErrorJSON("empty_slot",
			fmt.Sprintf("the %q slot has no key set — run `commitcraft ai key set --slot %s` first",
				target, target))
		return 1
	}
	if err := config.SaveEnvVar(config.EnvGroqActive, target); err != nil {
		printErrorJSON("config_error", err.Error())
		return 1
	}
	state.ActiveSlot = target
	printKeyState(state)
	return 0
}

// promptSlot asks which slot to set on stdin, defaulting to user on a blank
// line.
func promptSlot() string {
	fmt.Fprintf(os.Stderr, "Which slot? [user|ai] (default user): ")
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	if line == config.KeySlotAI {
		return config.KeySlotAI
	}
	if line == "" {
		return config.KeySlotUser
	}
	return line
}

// promptSecret reads a secret from stdin without echoing it when stdin is a
// terminal; falls back to a plain line read for piped/redirected input so
// scripted use still works.
func promptSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return line, nil
}
