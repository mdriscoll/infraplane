// Package executor provides a secure CLI command executor for running
// read-only cloud provider commands (gcloud, aws) with strict validation.
package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// maxOutputSize is the maximum bytes of stdout to capture (32KB).
const maxOutputSize = 32 * 1024

// CommandResult holds the output of a single command execution.
type CommandResult struct {
	Command  string `json:"command"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`
}

// Executor runs CLI commands with security validation.
type Executor struct {
	timeout time.Duration
}

// NewExecutor creates a new command executor with a default 30-second timeout.
func NewExecutor() *Executor {
	return &Executor{timeout: 30 * time.Second}
}

// dangerousPatterns are substrings that must NEVER appear in commands.
var dangerousPatterns = []string{
	"create", "delete", "destroy", "update", "deploy",
	"set-iam", "add-iam", "remove-iam",
	"import", "export",
	"rm ", " rm",
	"mv ", " mv",
	"--force", "--yes",
	"&&", "||", ";", "|", ">", "<", "`", "$(",
	"eval", "exec", "source",
}

// ValidateCommand checks that a command is safe to execute.
// It only allows read-only gcloud/aws commands.
func (e *Executor) ValidateCommand(cmd string) error {
	cmd = strings.TrimSpace(cmd)

	if cmd == "" {
		return fmt.Errorf("empty command")
	}

	// Must start with gcloud or aws
	if !strings.HasPrefix(cmd, "gcloud ") && !strings.HasPrefix(cmd, "aws ") {
		return fmt.Errorf("command must start with 'gcloud' or 'aws'")
	}

	// Check for dangerous patterns (shell injection, mutation commands)
	cmdLower := strings.ToLower(cmd)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmdLower, pattern) {
			return fmt.Errorf("command contains forbidden pattern %q", pattern)
		}
	}

	// For gcloud: must contain a read-only action
	if strings.HasPrefix(cmd, "gcloud ") {
		return validateGcloudCommand(cmd)
	}

	// For aws: must contain a read-only action
	if strings.HasPrefix(cmd, "aws ") {
		return validateAWSCommand(cmd)
	}

	return nil
}

func validateGcloudCommand(cmd string) error {
	parts := strings.Fields(cmd)
	hasReadAction := false
	for _, part := range parts {
		if part == "list" || part == "describe" {
			hasReadAction = true
			break
		}
	}
	if !hasReadAction {
		return fmt.Errorf("gcloud command must include 'list' or 'describe' action")
	}
	return nil
}

func validateAWSCommand(cmd string) error {
	parts := strings.Fields(cmd)
	hasReadAction := false
	for _, part := range parts {
		if strings.HasPrefix(part, "list-") ||
			strings.HasPrefix(part, "describe-") ||
			strings.HasPrefix(part, "get-") {
			hasReadAction = true
			break
		}
	}
	if !hasReadAction {
		return fmt.Errorf("aws command must include a list-*, describe-*, or get-* action")
	}
	return nil
}

// Execute runs a validated command and returns the result.
// Commands are executed directly (no shell) to prevent injection.
func (e *Executor) Execute(ctx context.Context, command string) CommandResult {
	if err := e.ValidateCommand(command); err != nil {
		return CommandResult{
			Command: command,
			Error:   fmt.Sprintf("validation failed: %s", err),
		}
	}

	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Split command into args — no shell interpretation
	parts := strings.Fields(command)
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Truncate large outputs
	stdoutStr := stdout.String()
	if len(stdoutStr) > maxOutputSize {
		stdoutStr = stdoutStr[:maxOutputSize]
	}

	result := CommandResult{
		Command: command,
		Stdout:  stdoutStr,
		Stderr:  stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Error = err.Error()
	}

	return result
}
