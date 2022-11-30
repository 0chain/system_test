package model

type StatusCallback struct{}

func (sc *StatusCallback) Started(allocationId string, filePath string, op int, totalBytes int) {}
func (sc *StatusCallback) InProgress(allocationId string, filePath string, op int, completedBytes int, data []byte) {
}
func (sc *StatusCallback) Error(allocationID string, filePath string, op int, err error) {}
func (sc *StatusCallback) Completed(allocationId string, filePath string, filename string, mimetype string, size int, op int) {
}
func (sc *StatusCallback) RepairCompleted(filesRepaired int) {}
