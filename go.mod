module github.com/0chain/system_test

go 1.16

require (
	github.com/0chain/gosdk v1.4.2
	github.com/herumi/bls-go-binary v1.0.1-0.20210830012634-a8e769d3b872
	github.com/kr/pretty v0.3.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20211209193657-4570a0811e8b
)

// temporary, for development
replace github.com/0chain/gosdk => ../gosdk
