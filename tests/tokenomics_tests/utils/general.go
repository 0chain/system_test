package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/api/util/test"
)

const (
	configPath = "./zbox_config.yaml"

	KB = 1024      // kilobyte
	MB = 1024 * KB // megabyte
	GB = 1024 * MB // gigabyte

	tokenUnit float64 = 1e+10
)

func EscapedTestName(t *test.SystemTest) string {
	replacer := strings.NewReplacer("/", "-", "\"", "-", ":", "-", "(", "-",
		")", "-", "<", "LESS_THAN", ">", "GREATER_THAN", "|", "-", "*", "-",
		"?", "-")
	return replacer.Replace(t.Name())
}

func CreateParams(params map[string]interface{}) string {
	var builder strings.Builder

	for k, v := range params {
		if v == nil {
			_, _ = builder.WriteString(fmt.Sprintf("--%s ", k))
		} else if reflect.TypeOf(v).String() == "bool" {
			_, _ = builder.WriteString(fmt.Sprintf("--%s=%v ", k, v))
		} else {
			_, _ = builder.WriteString(fmt.Sprintf("--%s %v ", k, v))
		}
	}
	return strings.TrimSpace(builder.String())
}

func IntToZCN(balance int64) float64 {
	return float64(balance) / tokenUnit
}

func UnitToZCN(unitCost float64, unit string) float64 {
	switch unit {
	case "SAS", "sas":
		unitCost /= 1e10
		return unitCost
	case "uZCN", "uzcn":
		unitCost /= 1e6
		return unitCost
	case "mZCN", "mzcn":
		unitCost /= 1e3
		return unitCost
	}
	return unitCost
}

func AssertOutputMatchesAllocationRegex(t *test.SystemTest, re *regexp.Regexp, str string) {
	match := re.FindStringSubmatch(str)
	require.True(t, len(match) > 0, "expected allocation to match regex", re, str)
}
