package model

import (
	"github.com/0chain/gosdk/core/transaction"
)

type StubStatusBar struct {
}

// Stub status bar used for committing actions

func (s *StubStatusBar) Started(allocationId, filePath string, op, totalBytes int) {
}
func (s *StubStatusBar) InProgress(allocationId, filePath string, op, completedBytes int, data []byte) {
}

func (s *StubStatusBar) Completed(allocationId, filePath string, filename string, mimetype string, size, op int) {
}

func (s *StubStatusBar) Error(allocationID, filePath string, op int, err error) {
}

func (s *StubStatusBar) CommitMetaCompleted(request, response string, txn *transaction.Transaction, err error) {
}

func (s *StubStatusBar) RepairCompleted(filesRepaired int) {
}
