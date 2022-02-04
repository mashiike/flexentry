package flexentry

import (
	"fmt"
	"strings"
)

func MergeEnv(base, other []string) []string {
	otherMap := envSliceToMap(other)
	return MergeEnvWithMap(base, otherMap)
}

func MergeEnvWithMap(base []string, otherMap map[string]string) []string {
	baseMap := envSliceToMap(base)
	for key, value := range otherMap {
		baseMap[key] = value
	}
	ret := make([]string, 0, len(baseMap))
	for key, value := range baseMap {
		ret = append(ret, fmt.Sprintf("%s=%s", key, value))
	}
	return ret
}

func envSliceToMap(base []string) map[string]string {
	ret := make(map[string]string, len(base))
	for _, env := range base {
		parts := strings.SplitN(env, "=", 2)
		var key, value string
		key = parts[0]
		if len(parts) >= 2 {
			value = parts[1]
		}
		ret[key] = value
	}
	return ret
}
