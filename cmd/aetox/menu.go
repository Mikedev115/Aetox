package main

// The first-launch menu: which provider, which model, how deep — asked once,
// on a keyboard, when nothing has chosen a model yet. Everything picked here
// is handed to the engine's own dials (run.go), which remember it the way the
// desktop's picker does.

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/oauth"

	"golang.org/x/term"
)

func formatModelModeLabel(providerName, modelName, thinkLevel string) string {
	status := model.ResolveStatus(providerName, modelName, nil)
	level := defaultThinkLevel(providerName, modelName, thinkLevel)
	if level == "" {
		return status
	}
	return fmt.Sprintf("%s(%s)", status, level)
}

func promptModelSelection(cfg config.Config, askThinkLevel bool) (string, string, string, string, string, bool) {
	reader := bufio.NewReader(os.Stdin)
	_, hasStoredPreference, prefErr := config.LoadModelPreference()
	if prefErr != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot read model preference: %v\n", prefErr)
	}

	providers := model.SupportedProviders()
	providerOptions := make([]string, 0, len(providers))
	for _, p := range providers {
		label := p
		if model.RequiresAPIKey(p) {
			label = model.FormatProviderMenuLabel(p, keyFor(p) != "")
		}
		providerOptions = append(providerOptions, label)
	}

	// Straight through, not a loop. This was a `for` whose every path returned,
	// so it read as "pick a provider, and come back here if that did not work"
	// while never coming back at all. The way back does not exist to restore:
	// pickModelForProvider returns a bare string with no room to say the user
	// backed out, so offering the choice again would be a feature, not a fix.
	// Removed rather than left promising something (staticcheck SA4004, §141).
	idx, ok := pickFromMenu(reader, "No model provider configured. Select one.", providerOptions, 0, "Use ↑/↓ then Enter.")
	if !ok {
		defaultProvider := providers[0]
		defaultModel := model.ResolveDefaultModel(defaultProvider, cfg.ModelBaseURL, keyFor(defaultProvider))
		return defaultProvider, defaultModel, "", cfg.ModelBaseURL, defaultThinkLevel(defaultProvider, defaultModel, cfg.ThinkLevel), false
	}
	provider := providers[idx]
	providerBaseURL := model.DefaultBaseURL(provider)
	if strings.TrimSpace(cfg.ModelBaseURL) != "" {
		providerBaseURL = strings.TrimSpace(cfg.ModelBaseURL)
	}

	key := keyFor(provider)

	// Needing credentials and taking a pasted key are two different facts,
	// and asking only the first one trapped anyone who picked Codex: it is
	// a ChatGPT subscription reached at chatgpt.com, the only key a user
	// could paste belongs to api.openai.com, and the loop below refuses an
	// empty line — so the menu demanded, forever, a credential that does
	// not exist. Sign-in is the way in; say so and carry on keyless.
	switch {
	case model.RequiresAPIKey(provider) && !model.AcceptsAPIKey(provider):
		if oauth.Has(provider) {
			fmt.Printf("Using the %s sign-in on this machine.\n", provider)
		} else {
			fmt.Printf("%s is a sign-in, not an API key. Run: aetox login %s\n", provider, provider)
		}
	case model.RequiresAPIKey(provider):
		if key == "" {
			if hasStoredPreference {
				fmt.Printf("No cached API key for %s.\n", provider)
			}
			for {
				fmt.Printf("API key for %s: ", provider)
				key = strings.TrimSpace(readLine(reader))
				if key != "" {
					break
				}
				fmt.Println("Missing API key. Try again.")
			}
		} else {
			fmt.Printf("Use existing API key for %s.\n", provider)
		}
	}

	selectedModel := pickModelForProvider(reader, provider, cfg.ModelName, providerBaseURL, key)
	selectedModel, selectedThinkLevel, parsedModelThink := parseModelWithThink(selectedModel)
	if !parsedModelThink {
		selectedThinkLevel = defaultThinkLevel(provider, selectedModel, cfg.ThinkLevel)
		if askThinkLevel {
			selectedThinkLevel = promptThinkLevelSelection(reader, provider, selectedModel, cfg.ThinkLevel)
		}
	}

	fmt.Printf("Selected: %s\n\n", formatModelModeLabel(provider, selectedModel, selectedThinkLevel))

	return provider, selectedModel, key, providerBaseURL, selectedThinkLevel, true
}

func defaultThinkLevel(provider, modelName, existing string) string {
	return model.NormalizeThinkingLevel(provider, modelName, existing)
}

func promptThinkLevelSelection(reader *bufio.Reader, provider, modelName, existing string) string {
	defaultLevel := defaultThinkLevel(provider, modelName, existing)
	if reader == nil {
		return defaultLevel
	}

	options := model.SupportedThinkingLevels(provider, modelName)
	if len(options) == 0 {
		return defaultLevel
	}
	defaultIndex := 0
	for i, option := range options {
		if option == defaultLevel {
			defaultIndex = i
			break
		}
	}

	idx, ok := pickFromMenu(reader, "Choose thinking level", options, defaultIndex, "Use ↑/↓ then Enter.")
	if !ok {
		return defaultLevel
	}
	return options[idx]
}

func pickModelForProvider(reader *bufio.Reader, provider, existing, baseURL, apiKey string) string {
	modelChoices, err := model.ModelChoicesWithEndpointAndAPIKey(provider, baseURL, apiKey)
	if err != nil || len(modelChoices) == 0 {
		modelChoices = model.ModelChoices(provider)
	}
	// Local providers carry no catalog default, so the first discovered model
	// is the default — modelChoices is already in hand, no second round trip.
	defaultModel := model.DefaultModel(provider)
	if defaultModel == "" && len(modelChoices) > 0 {
		defaultModel = modelChoices[0]
	}
	if existing != "" {
		defaultModel = existing
	}

	if len(modelChoices) == 0 {
		fmt.Printf("Model name for %s [%s] (or type custom): ", provider, defaultModel)
		if model := strings.TrimSpace(readLine(reader)); model != "" {
			return model
		}
		return defaultModel
	}

	options := append([]string{}, modelChoices...)
	// If current model is not in advertised list, keep it as a selectable default.
	if defaultModel != "" {
		foundDefault := false
		for _, m := range options {
			if m == defaultModel {
				foundDefault = true
				break
			}
		}
		if !foundDefault {
			options = append([]string{defaultModel}, options...)
		}
	}
	options = append(options, "custom model ...")
	defaultIndex := 0
	for i, m := range options {
		if i >= len(options)-1 {
			break
		}
		if m == defaultModel {
			defaultIndex = i
			break
		}
	}

	idx, ok := pickFromMenu(reader, fmt.Sprintf("Choose model for %s", provider), options, defaultIndex, "Use ↑/↓ then Enter.")
	if !ok {
		return defaultModel
	}

	if idx == len(options)-1 {
		fmt.Printf("Model name for %s [%s]: ", provider, defaultModel)
		if model := strings.TrimSpace(readLine(reader)); model != "" {
			return model
		}
		return defaultModel
	}

	return options[idx]
}

func pickFromMenu(reader *bufio.Reader, title string, options []string, defaultIndex int, hint string) (int, bool) {
	if len(options) == 0 {
		return 0, true
	}
	selected := defaultIndex
	if selected < 0 || selected >= len(options) {
		selected = 0
	}
	renderedLines := len(options) + 3
	interactiveMode := isInteractive()
	render := func() {
		fmt.Println()
		fmt.Println(title)
		for i, option := range options {
			prefix := "  "
			if i == selected {
				prefix = " >"
			}
			fmt.Printf("%s %s\n", prefix, option)
		}
		fmt.Println(hint)
	}
	redrawMenu := func() {
		if !interactiveMode {
			return
		}
		for i := 0; i < renderedLines; i++ {
			fmt.Print("\033[2K\r\033[F")
		}
	}
	clearMenu := func() {
		if !interactiveMode {
			return
		}
		for i := 0; i < renderedLines+1; i++ {
			fmt.Print("\033[2K\r\033[F")
		}
	}

	if !isInteractive() {
		fmt.Println(title)
		for i, option := range options {
			fmt.Printf("  %d) %s\n", i+1, option)
		}
		for {
			fmt.Printf("Select [1-%d]: ", len(options))
			input := strings.TrimSpace(readLine(reader))
			if input == "" {
				return selected, true
			}
			if input == "0" {
				return selected, true
			}
			for i := range options {
				if input == fmt.Sprint(i+1) {
					return i, true
				}
			}
			fmt.Println("Invalid selection.")
		}
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		// fallback: keep old behavior.
		return selectMenuUsingNumbers(reader, title, options, selected)
	}
	defer func() {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
	}()

	render()
	for {
		input, err := readSingleKey(reader)
		if err != nil {
			return selected, false
		}
		switch input {
		case keyMenuUp:
			selected--
			if selected < 0 {
				selected = len(options) - 1
			}
		case keyMenuDown:
			selected++
			if selected >= len(options) {
				selected = 0
			}
		case keyMenuEnter:
			clearMenu()
			return selected, true
		case keyMenuCancel:
			clearMenu()
			return selected, false
		}
		redrawMenu()
		render()
	}
}

const (
	keyMenuUp = iota + 1
	keyMenuDown
	keyMenuEnter
	keyMenuCancel
)

func readSingleKey(reader *bufio.Reader) (int, error) {
	b, err := reader.ReadByte()
	if err != nil {
		return 0, err
	}
	switch b {
	case 0x00:
		next, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		switch next {
		case 'H':
			return keyMenuUp, nil
		case 'P':
			return keyMenuDown, nil
		default:
			return 0, nil
		}
	case 0x1b:
		next, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		if next != '[' {
			return 0, nil
		}
		next, err = reader.ReadByte()
		if err != nil {
			return 0, err
		}
		switch next {
		case 'A':
			return keyMenuUp, nil
		case 'B':
			return keyMenuDown, nil
		default:
			return 0, nil
		}
	case 0x0d, 0x0a:
		return keyMenuEnter, nil
	case 0x03:
		return keyMenuCancel, nil
	default:
		return int(b), nil
	}
}

func selectMenuUsingNumbers(reader *bufio.Reader, title string, options []string, selected int) (int, bool) {
	for {
		fmt.Println(title)
		for i, option := range options {
			prefix := "  "
			if i == selected {
				prefix = " >"
			}
			fmt.Printf("%s %s\n", prefix, option)
		}
		fmt.Printf("Select [1-%d, Enter=default]: ", len(options))
		input := strings.TrimSpace(readLine(reader))
		if input == "" {
			return selected, true
		}
		if n, err := parseIndexSelection(input); err == nil {
			if n < 0 || n >= len(options) {
				fmt.Println("Invalid selection.")
				continue
			}
			return n, true
		}
		fmt.Println("Invalid selection.")
	}
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(strings.TrimSuffix(line, "\r\n"))
}

func parseIndexSelection(input string) (int, error) {
	value, err := strconv.Atoi(input)
	if err != nil {
		return 0, err
	}
	return value - 1, nil
}
