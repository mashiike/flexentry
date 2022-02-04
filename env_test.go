package flexentry_test

import (
	"fmt"
	"testing"

	"github.com/mashiike/flexentry"
	"github.com/stretchr/testify/require"
)

func TestMergeEnv(t *testing.T) {
	cases := []struct {
		left     []string
		right    []string
		excepted []string
	}{
		{
			right: []string{
				"HOGE=hoge",
			},
			excepted: []string{
				"HOGE=hoge",
			},
		},
		{
			left: []string{
				"FUGA=fuga",
			},
			right: []string{
				"HOGE=hoge",
			},
			excepted: []string{
				"FUGA=fuga",
				"HOGE=hoge",
			},
		},
		{
			left: []string{
				"FUGA=fuga",
				"PIYO=piyo",
			},
			right: []string{
				"HOGE=hoge",
				"PIYO=piyopiyo",
			},
			excepted: []string{
				"FUGA=fuga",
				"HOGE=hoge",
				"PIYO=piyopiyo",
			},
		},
		{
			left: []string{
				"FUGA=fuga",
				"PIYO=piyo",
			},
			right: []string{
				"HOGE=hoge",
				"PIYO=",
			},
			excepted: []string{
				"FUGA=fuga",
				"HOGE=hoge",
				"PIYO=",
			},
		},
		{
			left: []string{
				"FUGA=fuga",
				"PIYO=",
			},
			right: []string{
				"HOGE=hoge",
				`PIYO={"hoge":"hoge=hoge"}`,
			},
			excepted: []string{
				"FUGA=fuga",
				"HOGE=hoge",
				`PIYO={"hoge":"hoge=hoge"}`,
			},
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("case.%d", i), func(t *testing.T) {
			actual := flexentry.MergeEnv(c.left, c.right)
			require.ElementsMatch(t, c.excepted, actual)
		})
	}
}
