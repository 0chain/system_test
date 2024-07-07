package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestCompares0boxTablesWithSharder(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Compare 0box tables with sharder tables")

	t.RunSequentially("Compare Miner tables", func(t *test.SystemTest) {

		minersTable_Sharder, resp, err := apiClient.QueryDataFromSharder(t, "miner")
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())

		minersTable_0box, resp, err := zboxClient.QueryDataFrom0box(t, "miner")
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())

		require.Equal(t, len(minersTable_Sharder), len(minersTable_0box))
		// t.Logf("minersTable_Sharder: %+v", minersTable_Sharder)
		// t.Logf("minersTable_0box: %+v", minersTable_0box)

		var miners_Sharder, miners_0box map[string]map[string]interface{}
		miners_Sharder = make(map[string]map[string]interface{})
		miners_0box = make(map[string]map[string]interface{})
		for i := 0; i < len(minersTable_Sharder); i++ {
			m_sharder := minersTable_Sharder[i].(map[string]interface{})
			m_0box := minersTable_0box[i].(map[string]interface{})
			miners_Sharder[m_sharder["ID"].(string)] = m_sharder
			miners_0box[m_0box["id"].(string)] = m_0box
		}
		for k, v := range miners_Sharder {
			for k1, v1 := range v {
				if v1 != nil && miners_0box[k][k1] != nil {
					if v1 != miners_0box[k][k1] {
						t.Logf("id:%s and key:%s", k, k1)
					}
					require.Equal(t, v1, miners_0box[k][k1])
				}
			}

		}
	},
	)
}
