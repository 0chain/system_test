package tokenomics

const tokenUnit = 1e+10

func IntToZCN(num float64) int64 {
	return int64(num * tokenUnit)
}

func ZcnToInt(num float64) int64 {
	return int64(num / tokenUnit)
}
