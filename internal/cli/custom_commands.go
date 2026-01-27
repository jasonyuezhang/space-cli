package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/happy-sdk/space-cli/internal/dns"
	"github.com/happy-sdk/space-cli/pkg/config"
	"github.com/spf13/cobra"
)

// CustomCommandContext is passed to custom commands as JSON
type CustomCommandContext struct {
	WorkDir     string                       `json:"work_dir"`
	ProjectName string                       `json:"project_name"`
	Hash        string                       `json:"hash"`
	BaseDomain  string                       `json:"base_domain"`
	Command     string                       `json:"command"`
	Args        []string                     `json:"args"`
	Services    map[string]CustomServiceInfo `json:"services"`
}

// CustomServiceInfo contains service information for custom commands
type CustomServiceInfo struct {
	Name         string `json:"name"`
	DNSName      string `json:"dns_name"`
	InternalPort int    `json:"internal_port"`
	URL          string `json:"url"`
}

// findCustomCommand looks for a custom command in .space/commands/
func findCustomCommand(workDir, cmdName string) (string, error) {
	commandsDir := filepath.Join(workDir, ".space", "commands")

	// Check if commands directory exists
	if _, err := os.Stat(commandsDir); os.IsNotExist(err) {
		return "", nil
	}

	// Look for command with various extensions
	extensions := []string{"", ".sh", ".py", ".js", ".ts", ".go", ".rb", ".pl"}

	for _, ext := range extensions {
		cmdPath := filepath.Join(commandsDir, cmdName+ext)
		if info, err := os.Stat(cmdPath); err == nil {
			// Check if it's executable (for extensionless files) or has known extension
			if ext != "" || (info.Mode()&0111 != 0) {
				return cmdPath, nil
			}
		}
	}

	return "", nil
}

// getInterpreter returns the interpreter and args for a given file
func getInterpreter(filePath string) (string, []string) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".py":
		return "python3", nil
	case ".js":
		return "node", nil
	case ".ts":
		// Check for common TS runners
		if _, err := exec.LookPath("bun"); err == nil {
			return "bun", []string{"run"}
		}
		if _, err := exec.LookPath("tsx"); err == nil {
			return "tsx", nil
		}
		if _, err := exec.LookPath("ts-node"); err == nil {
			return "ts-node", nil
		}
		return "npx", []string{"tsx"}
	case ".go":
		return "go", []string{"run"}
	case ".rb":
		return "ruby", nil
	case ".pl":
		return "perl", nil
	case ".sh", "":
		return "sh", nil
	default:
		// Try to execute directly (might have shebang)
		return filePath, nil
	}
}

// runCustomCommand executes a custom command
func runCustomCommand(cmdPath, workDir string, args []string) error {
	// Load config for service info
	loader, err := config.NewLoader(workDir)
	if err != nil {
		return fmt.Errorf("failed to create config loader: %w", err)
	}

	cfg, _ := loader.Load() // Ignore error, config is optional

	// Build context
	hash := dns.GenerateDirectoryHash(workDir)
	ctx := CustomCommandContext{
		WorkDir:     workDir,
		ProjectName: filepath.Base(workDir),
		Hash:        hash,
		BaseDomain:  "space.local",
		Command:     filepath.Base(cmdPath),
		Args:        args,
		Services:    make(map[string]CustomServiceInfo),
	}

	if cfg != nil {
		if cfg.Project.Name != "" {
			ctx.ProjectName = cfg.Project.Name
		}
		for name, svc := range cfg.Services {
			dnsName := fmt.Sprintf("%s-%s.%s", name, hash, ctx.BaseDomain)
			ctx.Services[name] = CustomServiceInfo{
				Name:         name,
				DNSName:      dnsName,
				InternalPort: svc.Port,
				URL:          fmt.Sprintf("http://%s:%d", dnsName, svc.Port),
			}
		}
	}

	// Marshal context to JSON
	contextJSON, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	// Build environment
	env := os.Environ()
	env = append(env,
		"SPACE_WORKDIR="+ctx.WorkDir,
		"SPACE_PROJECT_NAME="+ctx.ProjectName,
		"SPACE_HASH="+ctx.Hash,
		"SPACE_BASE_DOMAIN="+ctx.BaseDomain,
	)

	for name, svc := range ctx.Services {
		prefix := "SPACE_SERVICE_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
		env = append(env, prefix+"_DNS_NAME="+svc.DNSName)
		env = append(env, fmt.Sprintf("%s_PORT=%d", prefix, svc.InternalPort))
		env = append(env, prefix+"_URL="+svc.URL)
	}

	// Get interpreter
	interpreter, interpreterArgs := getInterpreter(cmdPath)

	// Build command
	var cmdArgs []string
	if interpreter == cmdPath {
		// Direct execution
		cmdArgs = args
	} else {
		// Interpreter execution
		cmdArgs = append(interpreterArgs, cmdPath)
		cmdArgs = append(cmdArgs, args...)
	}

	cmd := exec.Command(interpreter, cmdArgs...)
	cmd.Dir = workDir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Pass context via stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Write context JSON to stdin and close
	_, _ = stdin.Write(contextJSON)
	_ = stdin.Close()

	return cmd.Wait()
}

// listCustomCommands returns all available custom commands
func listCustomCommands(workDir string) []string {
	commandsDir := filepath.Join(workDir, ".space", "commands")

	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var commands []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip hidden files and READMEs
		if strings.HasPrefix(name, ".") || strings.EqualFold(name, "README.md") {
			continue
		}

		// Remove extension to get command name
		cmdName := strings.TrimSuffix(name, filepath.Ext(name))

		if !seen[cmdName] {
			seen[cmdName] = true
			commands = append(commands, cmdName)
		}
	}

	sort.Strings(commands)
	return commands
}

// newRunCommand creates the 'run' command for executing custom commands
func newRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <command> [args...]",
		Short: "Run a custom command from .space/commands/",
		Long: `Execute a custom command defined in .space/commands/.

Commands can be written in any language:
  - Shell (.sh or no extension)
  - Python (.py)
  - JavaScript (.js)
  - TypeScript (.ts)
  - Go (.go)
  - Ruby (.rb)

Commands receive context via:
  - Environment variables (SPACE_WORKDIR, SPACE_HASH, SPACE_SERVICE_*, etc.)
  - JSON on stdin with full project context

Example:
  space run db-seed
  space run deploy --env staging`,
		Args:               cobra.MinimumNArgs(1),
		DisableFlagParsing: true, // Pass all flags to the custom command
		RunE: func(cmd *cobra.Command, args []string) error {
			workDir := Workdir
			if workDir == "." {
				var err error
				workDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get working directory: %w", err)
				}
			}
			workDir, _ = filepath.Abs(workDir)

			cmdName := args[0]
			cmdArgs := args[1:]

			// Find the command
			cmdPath, err := findCustomCommand(workDir, cmdName)
			if err != nil {
				return err
			}

			if cmdPath == "" {
				// List available commands
				available := listCustomCommands(workDir)
				if len(available) == 0 {
					return fmt.Errorf("command %q not found\n\nNo custom commands found. Create commands in .space/commands/", cmdName)
				}
				return fmt.Errorf("command %q not found\n\nAvailable commands:\n  %s", cmdName, strings.Join(available, "\n  "))
			}

			return runCustomCommand(cmdPath, workDir, cmdArgs)
		},
	}

	// Add list subcommand
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List available custom commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			workDir := Workdir
			if workDir == "." {
				var err error
				workDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get working directory: %w", err)
				}
			}
			workDir, _ = filepath.Abs(workDir)

			commands := listCustomCommands(workDir)
			if len(commands) == 0 {
				fmt.Println("No custom commands found.")
				fmt.Println("\nCreate commands in .space/commands/")
				fmt.Println("Supported: .sh, .py, .js, .ts, .go, .rb")
				return nil
			}

			fmt.Println("Available custom commands:")
			for _, c := range commands {
				cmdPath, _ := findCustomCommand(workDir, c)
				ext := filepath.Ext(cmdPath)
				if ext == "" {
					ext = "shell"
				} else {
					ext = ext[1:] // Remove leading dot
				}
				fmt.Printf("  %s (%s)\n", c, ext)
			}
			return nil
		},
	})

	return cmd
}

// HandleUnknownCommand checks if an unknown command is a custom command
// Returns true if it was handled, false otherwise
func HandleUnknownCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}

	workDir := Workdir
	if workDir == "." {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return false
		}
	}
	workDir, _ = filepath.Abs(workDir)

	cmdName := args[0]
	cmdPath, err := findCustomCommand(workDir, cmdName)
	if err != nil || cmdPath == "" {
		return false
	}

	// Found a custom command, execute it
	if err := runCustomCommand(cmdPath, workDir, args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return true
}
