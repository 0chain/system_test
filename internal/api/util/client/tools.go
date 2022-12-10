package client

func IsGraphGreater(src1, src2 []int) bool {
	var src1Sum int
	for _, v := range src1 {
		src1Sum += v
	}

	var src2Sum int
	for _, v := range src2 {
		src2Sum += v
	}
	return src1Sum > src2Sum
}

func IsGraphLess(src1, src2 []int) bool {
	var src1Sum int
	for _, v := range src1 {
		src1Sum += v
	}

	var src2Sum int
	for _, v := range src2 {
		src2Sum += v
	}
	return src1Sum < src2Sum
}
