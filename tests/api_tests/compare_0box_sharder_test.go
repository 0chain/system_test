package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/currency"
	"github.com/stretchr/testify/require"
)

func ParseToTimeIfValid(val interface{}) interface{} {
	// check if val is a time.Time as a string
	valStr, ok := val.(string)
	if ok {
		res, err := time.Parse(time.RFC3339Nano, valStr)
		if err == nil {
			return res
		}
	}
	// Parsing failed, v1Str is not a time string
	return val
}

func CompareEntityTables(t *test.SystemTest, entity string, id_sharder string, id_0box string) {

	entityTable_Sharder, resp, err := apiClient.QueryDataFromSharder(t, entity)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())

	entityTable_0box, resp, err := zboxClient.QueryDataFrom0box(t, entity)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())

	if entity == "user" {
		entityTable_SharderCopy := entityTable_Sharder
		entityTable_Sharder = entityTable_Sharder[:0]
		// t.Logf("entityTable_Sharder: %+v", entityTable_Sharder)

		for i := 0; i < len(entityTable_SharderCopy); i++ {
			// t.Logf("entityTable_SharderCopy[i].(map[string]interface{})[\"round\"]: %+v", entityTable_SharderCopy[i].(map[string]interface{})["round"])
			temp_copy := entityTable_SharderCopy[i].(map[string]interface{})
			val := temp_copy["round"]
			if val != nil {
				switch val.(type) {
				case float64:
					if val.(float64) != 0 {
						entityTable_Sharder = append(entityTable_Sharder, temp_copy)
					}
				case int:
					if val.(int) != 0 {
						entityTable_Sharder = append(entityTable_Sharder, temp_copy)
					}
				case int64:
					if val.(int64) != 0 {
						entityTable_Sharder = append(entityTable_Sharder, temp_copy)
					}
				}
			}
		}
	}
	require.Equal(t, len(entityTable_Sharder), len(entityTable_0box))
	// t.Logf("entityTable_Sharder: %+v", entityTable_Sharder)
	// t.Logf("entityTable_0box: %+v", entityTable_0box)

	var entity_Sharder, entity_0box map[string]map[string]interface{}
	entity_Sharder = make(map[string]map[string]interface{})
	entity_0box = make(map[string]map[string]interface{})
	for i := 0; i < len(entityTable_Sharder); i++ {
		e_sharder := entityTable_Sharder[i].(map[string]interface{})
		e_0box := entityTable_0box[i].(map[string]interface{})
		entity_Sharder[e_sharder[id_sharder].(string)] = e_sharder
		entity_0box[e_0box[id_0box].(string)] = e_0box
	}
	for k, v := range entity_Sharder {
		for k1, val1 := range v {
			if val1 != nil && entity_0box[k][k1] != nil {
				// how to know if v1 is a time.Time as a stringaf
				v1 := ParseToTimeIfValid(val1)
				switch v1.(type) {
				case float64:
					require.InDelta(t, v1.(float64), entity_0box[k][k1].(float64), 0.05*v1.(float64), "Float64 values are not equal for field %s", k1)
				case string:
					require.Equal(t, v1.(string), entity_0box[k][k1].(string))
				case bool:
					require.Equal(t, v1.(bool), entity_0box[k][k1].(bool))
				case int:
					require.InDelta(t, v1.(int), entity_0box[k][k1].(int), 0.05*float64(v1.(int)))
				case int64:
					require.InDelta(t, v1.(int64), entity_0box[k][k1].(int64), 0.05*float64(v1.(int64)))
				case uint64:
					require.InDelta(t, v1.(uint64), entity_0box[k][k1].(uint64), 0.05*float64(v1.(uint64)))
				case time.Time:
					t1 := v1.(time.Time)
					entity_0box[k][k1] = ParseToTimeIfValid(entity_0box[k][k1])
					t2 := entity_0box[k][k1].(time.Time)
					diff := t1.Sub(t2)
					require.LessOrEqual(t, diff.Seconds(), 1000.0, "Time difference is more than 1000 seconds")
					require.GreaterOrEqual(t, diff.Seconds(), -1000.0, "Time difference is more than -1000 seconds")
				case currency.Coin:
					t1, err1 := (v1.(currency.Coin)).Int64()
					t2, err2 := entity_0box[k][k1].(currency.Coin).Int64()
					require.NoError(t, err1)
					require.NoError(t, err2)
					require.InDelta(t, t1, t2, 0.05*float64(t1))

				}

			}
		}

	}
}

func TestCompares0boxTablesWithSharder(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Compare 0box tables with sharder tables")

	t.RunSequentially("Compare Miner tables", func(t *test.SystemTest) {
		CompareEntityTables(t, "miner", "ID", "id")
	})
	t.RunSequentially("Compare Blobber tables", func(t *test.SystemTest) {
		CompareEntityTables(t, "blobber", "ID", "id")
	})
	t.RunSequentially("Compare Sharder tables", func(t *test.SystemTest) {
		CompareEntityTables(t, "sharder", "ID", "id")
	})
	t.RunSequentially("Compare Validator tables", func(t *test.SystemTest) {
		CompareEntityTables(t, "validator", "ID", "id")
	})
	t.RunSequentially("Compare Authorizer tables", func(t *test.SystemTest) {
		CompareEntityTables(t, "authorizer", "ID", "id")
	})
	t.RunSequentially("Compare ProviderRewards tables", func(t *test.SystemTest) {
		CompareEntityTables(t, "provider_rewards", "provider_id", "provider_id")
	})
	t.RunSequentially("Compare User tables", func(t *test.SystemTest) {
		CompareEntityTables(t, "user", "txn_hash", "txn_hash")
	})

}
