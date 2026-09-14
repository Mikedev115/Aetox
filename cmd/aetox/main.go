package main

// aetox — the console screen of the engine (§248): the same engine the
// desktop window runs, in this process, at the coding desk. `aetox chat
// "goal"` is one turn and the answer on stdout; `aetox` alone is a line loop.
//
// This file is the command line — flags, the first-launch menu, the sign-in
// subcommand — and nothing of the session; run.go is the session and
// screen.go is what the engine sees of the terminal. The agent loop this
// command used to build for itself (cognitive.NewAgent on a prompt of its
// own, with its own registry and its own MCP wiring) is gone: it was a second
// harness wearing the app's name, without the desk, the memory or the skills
// the app has, and the only reason it survived was that nothing measured it.

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Mikedev115/Aetox/internal/command"
	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/proc"
	"github.com/Mikedev115/Aetox/internal/safety"
	"github.com/Mikedev115/Aetox/internal/think"
	"github.com/Mikedev115/Aetox/internal/version"
)

func parseModelWithThink(raw string) (string, string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", "", false
	}

	openIdx := strings.LastIndex(value, "(")
	closeIdx := strings.LastIndex(value, ")")
	if openIdx < 0 || closeIdx < 0 || closeIdx != len(value)-1 || closeIdx <= openIdx+1 {
		return value, "", false
	}

	inner := strings.TrimSpace(value[openIdx+1 : closeIdx])
	modelName := strings.TrimSpace(value[:openIdx])
	if modelName == "" || inner == "" {
		return value, "", false
	}

	normalized, err := think.ParseLevel(inner)
	if err != nil {
		return value, "", false
	}

	return modelName, string(normalized), true
}

func main() {
	// Children (MCP servers, shell tools) die with the process, even on
	// force-kill. See ARCHITECTURE.md §24.5.
	proc.KillTreeOnExit()
	setUTF8Console()

	// Sign-in is a subcommand, intercepted before the flag parser: everything
	// below assumes it is starting a session, and `aetox login copilot` is not
	// a session. Returns false for every other invocation.
	if handled, code := runAuthCommand(os.Args[1:]); handled {
		os.Exit(code)
	}
	// `aetox design` likewise: a linter run, not a session (design.go).
	if handled, code := runDesignCommand(os.Args[1:]); handled {
		os.Exit(code)
	}
	// `aetox skill lint` too: the shelf's writing rules as a check (skilllint.go).
	if handled, code := runSkillLintCommand(os.Args[1:]); handled {
		os.Exit(code)
	}

	// Install the cached model table before anything asks what a model can do.
	// The first-launch menu below reads thinking depths from it; the engine
	// installs it again for itself at startup. Reads a file, never the network.
	if root, err := config.DataRoot(); err == nil {
		model.InstallCachedCatalog(root)
	}

	providerUsageHint := "model provider (" + strings.Join(model.SupportedProviders(), "|") + ")"

	var (
		rootPath      string
		modelProvider string
		modelName     string
		modelAPIKey   string
		modelBaseURL  string
		thinkLevel    string
		approvalMode  string
		reportPath    string
		legacyYes     bool
		showVersion   bool
		showHelp      bool
	)

	argsWithoutGlobal, argsForIntent, preParseErr := preparseGlobalFlags(os.Args[1:])
	if preParseErr != nil {
		fmt.Fprintf(os.Stderr, "invalid flags: %v\n", preParseErr)
		os.Exit(2)
	}

	flags := flag.NewFlagSet("aetox", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&rootPath, "root", "", "project folder (default: current directory)")
	flags.StringVar(&modelProvider, "model-provider", "", providerUsageHint)
	flags.StringVar(&modelName, "model-name", "", "model name or model(think-level)")
	flags.StringVar(&modelAPIKey, "model-api-key", "", "model API key; fallback to the store or the provider's env var when empty")
	flags.StringVar(&modelBaseURL, "model-base-url", "", "override base URL for the model provider")
	flags.StringVar(&thinkLevel, "think", "", "thinking level (model/provider specific)")
	flags.StringVar(&approvalMode, "approval", "", "approval mode: ask, unsafe-only, or full-access")
	flags.StringVar(&reportPath, "report", "", "append one JSON line per turn to this file (rounds, tokens, cost, seconds, tools)")
	flags.BoolVar(&legacyYes, "yes", false, "same as --approval full-access")
	flags.BoolVar(&showVersion, "version", false, "print version")
	flags.BoolVar(&showHelp, "help", false, "print usage")
	_ = flags.Bool("h", false, "help alias")
	_ = flags.Bool("v", false, "version alias")
	_ = flags.Parse(argsWithoutGlobal)

	providerExplicit := strings.TrimSpace(modelProvider) != ""
	thinkLevelExplicit := strings.TrimSpace(thinkLevel) != ""
	if thinkLevelExplicit {
		parsedThinkLevel, err := think.ParseLevel(thinkLevel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid flags: %v\n", err)
			os.Exit(2)
		}
		thinkLevel = string(parsedThinkLevel)
	}
	modelNameFromFlag, parsedThinkLevel, modelNameHasThink := parseModelWithThink(modelName)
	if modelNameHasThink && !thinkLevelExplicit {
		modelName = modelNameFromFlag
		thinkLevel = parsedThinkLevel
	}

	if showVersion {
		fmt.Printf("aetox version %s\n%s\n", version.Current, version.Credit)
		return
	}
	if showHelp {
		printUsage()
		return
	}

	intent := command.ParseArgs(argsForIntent)
	switch intent.Mode {
	case command.ModeHelp:
		printUsage()
		return
	case command.ModeVersion:
		fmt.Printf("aetox version %s\n%s\n", version.Current, version.Credit)
		return
	case command.ModeInteractive, command.ModeOnce:
	default:
		printUsage()
		os.Exit(2)
	}

	// A key on the command line is for the provider on the command line, and
	// for nothing picked later from a menu. Every other key comes from the
	// store at the moment it is needed (keyFor) — config carries none (§248).
	if strings.TrimSpace(modelAPIKey) != "" {
		flagAPIKey, flagAPIKeyProvider = strings.TrimSpace(modelAPIKey), model.NormalizeProvider(modelProvider)
	}

	d := dials{
		Root:     rootPath,
		Provider: modelProvider,
		Model:    modelName,
		BaseURL:  modelBaseURL,
		Think:    thinkLevel,
		Report:   reportPath,
	}
	switch {
	case strings.TrimSpace(approvalMode) != "":
		d.Approval = string(safety.NormalizeApprovalMode(approvalMode))
	case legacyYes:
		d.Approval = string(safety.ApprovalFullAccess)
	}

	// First launch on a keyboard, nothing chosen anywhere yet: ask. A script
	// gets the engine's fallback and the warning that names it.
	_, hasStoredPreference, prefErr := config.LoadModelPreference()
	if prefErr != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot read model preference: %v\n", prefErr)
	}
	if intent.Mode == command.ModeInteractive && isInteractive() && !providerExplicit && !hasStoredPreference {
		provider, chosenModel, key, baseURL, chosenThink, ok := promptModelSelection(config.Config{
			ModelName:    modelName,
			ModelBaseURL: modelBaseURL,
			ThinkLevel:   thinkLevel,
		}, !thinkLevelExplicit)
		if ok {
			rememberKey(provider, key)
			d.Provider, d.Model, d.BaseURL = provider, chosenModel, baseURL
			if !thinkLevelExplicit {
				d.Think = chosenThink
			}
		}
	}

	c, err := startConsole(d, os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "runtime init failed: %v\n", err)
		os.Exit(1)
	}
	defer c.close()

	// One shot asked for an answer and the engine fell back to the built-in
	// provider, which answers every question with "connect a model": that is
	// a failure to a script, not an answer. In the line loop the warning was
	// printed and the person can /provider their way out.
	oneShot := intent.Mode == command.ModeOnce || !isInteractive()
	if info := c.e.GetModelInfo(); info.Warning != "" && !oneShot {
		fmt.Fprintf(os.Stderr, "warning: %s\n", info.Warning)
	} else if oneShot && info.Warning != "" && !strings.EqualFold(info.Provider, "aetox") {
		fmt.Fprintf(os.Stderr, "no model is answering: %s\n", info.Warning)
		c.close()
		os.Exit(2)
	}

	switch intent.Mode {
	case command.ModeInteractive:
		if !isInteractive() {
			// `echo goal | aetox` — the terminal is the message.
			text, ok := <-c.lines
			if !ok || strings.TrimSpace(text) == "" {
				printUsage()
				os.Exit(2)
			}
			if err := c.turn(text); err != nil {
				fmt.Fprintf(os.Stderr, "Chat failed: %v\n", err)
				c.close()
				os.Exit(1)
			}
			return
		}
		if err := c.interactive(); err != nil {
			fmt.Fprintf(os.Stderr, "interactive chat failed: %v\n", err)
			c.close()
			os.Exit(1)
		}
	case command.ModeOnce:
		if err := c.turn(intent.Message); err != nil {
			fmt.Fprintf(os.Stderr, "Chat failed: %v\n", err)
			c.close()
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  aetox [flags] [goal...]")
	fmt.Println("  aetox chat \"goal\"       run one turn at the coding desk, answer on stdout, exit")
	fmt.Println("  aetox                    interactive mode")
	fmt.Println("  aetox login <provider>   sign in (codex, copilot, ...)")
	fmt.Println("  aetox help               show this help")
	fmt.Println("Flags:")
	fmt.Printf("  --model-provider: %s\n", strings.Join(model.SupportedProviders(), "|"))
	fmt.Println("  --model-name <model[(think-level)]> optional; the remembered choice when omitted")
	fmt.Println("  --model-api-key <key>        fallback: the store, then the provider's env var")
	fmt.Println("  --model-base-url <url>       custom endpoint for the provider")
	fmt.Println("  --think <level>              thinking level (model/provider specific)")
	fmt.Println("  --approval <mode>            ask, unsafe-only, full-access (default: the remembered choice)")
	fmt.Println("  --yes                        same as --approval full-access")
	fmt.Println("  --root <dir>                 project folder (default: current directory)")
	fmt.Println("  --report <file>              append one JSON line per turn: rounds, tokens, cost, seconds, tools")
	fmt.Println("  --version                    print version")
	fmt.Println("The session always sits at the coding desk; logs go to the app's data folder (AETOX_DATA_ROOT to move it).")
}

func isInteractive() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func preparseGlobalFlags(rawArgs []string) ([]string, []string, error) {
	global := make([]string, 0, len(rawArgs))
	remaining := make([]string, 0, len(rawArgs))

	isValueFlag := func(arg string) bool {
		switch arg {
		case "--root", "--model-provider", "--model-name", "--model-api-key", "--model-base-url", "--think", "--approval", "--report":
			return true
		}
		return false
	}

	isBoolFlag := func(arg string) bool {
		switch arg {
		case "--yes", "--version", "--help", "-v", "-h":
			return true
		}
		return false
	}

	for idx := 0; idx < len(rawArgs); idx++ {
		raw := strings.TrimSpace(rawArgs[idx])
		if raw == "--" {
			remaining = append(remaining, raw)
			if idx+1 < len(rawArgs) {
				remaining = append(remaining, rawArgs[idx+1:]...)
			}
			break
		}

		if !strings.HasPrefix(raw, "--") && !(raw == "-h" || raw == "-v") {
			remaining = append(remaining, raw)
			continue
		}

		if strings.Contains(raw, "=") {
			nameValue := strings.SplitN(raw, "=", 2)
			name := strings.ToLower(strings.TrimSpace(nameValue[0]))
			value := ""
			if len(nameValue) > 1 {
				value = nameValue[1]
			}
			if isValueFlag(name) {
				global = append(global, name, value)
				continue
			}
			if isBoolFlag(name) {
				global = append(global, name)
				continue
			}
			remaining = append(remaining, raw)
			continue
		}

		if isBoolFlag(raw) {
			global = append(global, raw)
			continue
		}

		if isValueFlag(raw) {
			if idx+1 >= len(rawArgs) {
				return nil, nil, fmt.Errorf("flag %s requires a value", raw)
			}
			global = append(global, raw, rawArgs[idx+1])
			idx++
			continue
		}

		remaining = append(remaining, raw)
	}

	return global, remaining, nil
}
