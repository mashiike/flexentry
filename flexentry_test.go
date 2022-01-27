package flexentry_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/mashiike/flexentry"
	"github.com/stretchr/testify/require"
)

func TestEntrypointDetectCommand(t *testing.T) {
	cases := []struct {
		preAction  func()
		postAction func()
		event      flexentry.Event
		expected   []string
	}{
		{
			event:    "echo hoge",
			expected: []string{"echo hoge"},
		},
		{
			preAction: func() {
				os.Setenv("FLEXENTRY_COMMAND", "echo hoge")
			},
			postAction: func() {
				os.Unsetenv("FLEXENTRY_COMMAND")
			},
			expected: []string{"echo hoge"},
		},
		{
			event: map[string]interface{}{
				"cmd": "echo hoge",
			},
			expected: []string{"echo hoge"},
		},
		{
			preAction: func() {
				os.Setenv("FLEXENTRY_COMMAND_JQ_EXPR", ".cmd2")
			},
			postAction: func() {
				os.Unsetenv("FLEXENTRY_COMMAND_JQ_EXPR")
			},
			event: map[string]interface{}{
				"cmd":  "echo hoge",
				"cmd2": "echo fuga",
			},
			expected: []string{"echo fuga"},
		},
		{
			event:    []string{"echo", "hoge"},
			expected: []string{"echo", "hoge"},
		},
		{
			preAction: func() {
				os.Setenv("FLEXENTRY_COMMAND_JQ_EXPR", ".cmd | ..")
			},
			postAction: func() {
				os.Unsetenv("FLEXENTRY_COMMAND_JQ_EXPR")
			},
			event: map[string]interface{}{
				"cmd": []interface{}{"echo", 1},
			},
			expected: []string{"echo", "1"},
		},
		{
			event: map[string]interface{}{
				"cmd": []string{"echo", "fuga"},
			},
			expected: []string{"echo", "fuga"},
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case.%d", i), func(t *testing.T) {
			if c.preAction != nil {
				c.preAction()
			}
			if c.postAction != nil {
				defer c.postAction()
			}
			e := &flexentry.Entrypoint{}
			actual, err := e.DetectCommand(context.Background(), c.event)
			require.NoError(t, err)
			require.EqualValues(t, c.expected, actual)
		})
	}
}
