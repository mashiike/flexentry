package flexentry

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/handlename/ssmwrap"
)

type ExecuteOption struct {
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Environ []string
}
type Executer interface {
	Execute(ctx context.Context, opt *ExecuteOption, commands ...string) error
}

type ShellExecuter struct {
	shell     string
	shellArgs []string
}

func NewShellExecuter() *ShellExecuter {
	shell := "sh"
	if s := os.Getenv("FLEXENTRY_SHELL"); s != "" {
		shell = s
	}
	shellArgs := []string{"-c"}
	if sArgs := os.Getenv("FLEXENTRY_SHELL_ARGS"); sArgs != "" {
		shellArgs = strings.Split(sArgs, " ")
	}
	return &ShellExecuter{
		shell:     shell,
		shellArgs: shellArgs,
	}
}

func (e *ShellExecuter) Execute(ctx context.Context, opt *ExecuteOption, commands ...string) error {
	args := make([]string, 0, len(e.shellArgs)+len(commands))
	args = append(args, e.shellArgs...)
	if os.Getenv("FLEXENTRY_QUOTE_COMMAND") != "" {
		args = append(args, `"`+strings.Join(commands, " ")+`"`)
	} else {
		args = append(args, strings.Join(commands, " "))
	}

	log.Printf("[debug] $%s %s", e.shell, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, e.shell, args...)
	cmd.Env = os.Environ()
	p, _ := cmd.StdinPipe()
	var stdin io.Reader
	if opt != nil {
		stdin = opt.Stdin
		cmd.Stderr = opt.Stderr
		cmd.Stdout = opt.Stdout
		cmd.Env = MergeEnv(cmd.Env, opt.Environ)
	}
	go func() {
		defer p.Close()
		if stdin == nil {
			return
		}
		if _, err := io.Copy(p, stdin); err != nil {
			log.Println("[warn] failed to write stdinPipe:", err)
		}
	}()
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (e *ShellExecuter) Clone() *ShellExecuter {
	cloned := *e
	return &cloned
}

func (e *ShellExecuter) SetShell(shell string, shellArgs []string) *ShellExecuter {
	cloned := e.Clone()
	cloned.shell = shell
	cloned.shellArgs = make([]string, len(shellArgs))
	copy(cloned.shellArgs, shellArgs)
	return cloned
}

type SSMWrapExecuter struct {
	Executer

	mu              sync.Mutex
	lastExported    time.Time
	ssmCacheExpires time.Duration
}

func NewSSMWrapExecuter(executer Executer, cacheExpires time.Duration) *SSMWrapExecuter {
	return &SSMWrapExecuter{
		Executer:        executer,
		ssmCacheExpires: cacheExpires,
	}
}

func (e *SSMWrapExecuter) exportEnvWithCache() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.lastExported.IsZero() || e.lastExported.Before(time.Now().Add(-1*e.ssmCacheExpires)) {
		defer func() {
			e.lastExported = time.Now()
		}()
		return e.exportEnv()
	}
	log.Printf("[debug] exportEnv skipped. last exported at %s", e.lastExported.Format(time.RFC3339))
	return nil
}

func (e *SSMWrapExecuter) exportEnv() error {
	if paths := os.Getenv("SSMWRAP_PATHS"); paths == "" {
		return nil
	} else {
		if err := ssmwrap.Export(ssmwrap.ExportOptions{
			Paths:   strings.Split(paths, ","),
			Retries: 3,
		}); err != nil {
			return fmt.Errorf("failed to fetch values from SSM paths %s: %w", paths, err)
		}
		log.Printf("[debug] exportEnv from SSMWRAP_PATHS=%s", paths)
	}
	return nil
}

func (e *SSMWrapExecuter) Execute(ctx context.Context, opt *ExecuteOption, commands ...string) error {
	if err := e.exportEnvWithCache(); err != nil {
		return err
	}
	return e.Executer.Execute(ctx, opt, commands...)
}
