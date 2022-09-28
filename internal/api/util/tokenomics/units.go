package tokenomics

const tokenUnit = 1e+10

func IntToZCN(num float64) *int64 {
	result := int64(num * tokenUnit)
	return &result
}

func ZcnToInt(num float64) int64 {
	return int64(num / tokenUnit)
}
