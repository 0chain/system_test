package model

import (
	"fmt"
	"github.com/0chain/gosdk/core/transaction"
)

func (s *StubStatusBar) Started(allocationId, filePath string, op int, totalBytes int) {
}
func (s *StubStatusBar) InProgress(allocationId, filePath string, op int, completedBytes int, data []byte) {
}

func (s *StubStatusBar) Completed(allocationId, filePath string, filename string, mimetype string, size int, op int) {
}

func (s *StubStatusBar) Error(allocationID string, filePath string, op int, err error) {
}

func (s *StubStatusBar) CommitMetaCompleted(request, response string, txn *transaction.Transaction, err error) {
	fmt.Println(response, err)
}

func (s *StubStatusBar) RepairCompleted(filesRepaired int) {
}

type StubStatusBar struct {
}
