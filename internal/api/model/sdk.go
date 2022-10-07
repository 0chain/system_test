package model

import (
	"github.com/0chain/gosdk/core/transaction"
)

// Stub status bar used for commiting actions

func (s *StubStatusBar) Started(allocationId, filePath string, op int, totalBytes int) {
}
func (s *StubStatusBar) InProgress(allocationId, filePath string, op int, completedBytes int, data []byte) {
}

func (s *StubStatusBar) Completed(allocationId, filePath string, filename string, mimetype string, size int, op int) {
}

func (s *StubStatusBar) Error(allocationID string, filePath string, op int, err error) {
}

func (s *StubStatusBar) CommitMetaCompleted(request, response string, txn *transaction.Transaction, err error) {
}

func (s *StubStatusBar) RepairCompleted(filesRepaired int) {
}

type StubStatusBar struct {
}
