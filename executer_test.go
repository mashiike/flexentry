package flexentry_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mashiike/flexentry"
	"github.com/stretchr/testify/require"
)

func TestExecuter(t *testing.T) {
	testShell := "sh"
	if s := os.Getenv("FLEXENTRY_TEST_SHELL"); s != "" {
		testShell = s
	}
	testShellArgs := "-c"
	if args := os.Getenv("FLEXENTRY_TEST_SHELL_ARGS"); args != "" {
		testShellArgs = args
	}
	testShellExecuter := flexentry.NewShellExecuter().
		SetShell(testShell, strings.Split(testShellArgs, " "))

	cases := []struct {
		commands       []string
		stdin          []byte
		timeout        time.Duration
		exceptedErr    string
		exceptedStderr string
		exceptedStdout string
	}{
		{
			commands:       []string{"echo hoge"},
			exceptedStdout: "hoge\n",
		},
		{
			commands:       []string{"hoge"},
			exceptedErr:    "exit status 127",
			exceptedStderr: "sh: 1: hoge: not found\n",
		},
		{
			commands:    []string{"sleep 30"},
			timeout:     50 * time.Millisecond,
			exceptedErr: "signal: killed",
		},
	}
	for _, c := range cases {
		t.Run(strings.Join(c.commands, "_"), func(t *testing.T) {
			stdin := bytes.NewReader(c.stdin)
			var stdout, stderr bytes.Buffer
			executer := testShellExecuter
			timeout := 1 * time.Minute
			if c.timeout != 0 {
				timeout = c.timeout
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			err := executer.Execute(ctx, flexentry.Pipe{
				Stdin:  stdin,
				Stdout: &stdout,
				Stderr: &stderr,
			}, c.commands...)
			if c.exceptedErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, c.exceptedErr)
			}
			require.EqualValues(t, c.exceptedStdout, stdout.String(), "stdout")
			require.EqualValues(t, c.exceptedStderr, stderr.String(), "stderr")

		})
	}
}
