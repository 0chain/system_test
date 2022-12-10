package model

type StatusCallback struct{}

func (sc *StatusCallback) Started(allocationId, filePath string, op, totalBytes int) {
}
func (sc *StatusCallback) InProgress(allocationId, filePath string, op, completedBytes int, data []byte) {
}
func (sc *StatusCallback) Error(allocationID, filePath string, op int, err error) {}
func (sc *StatusCallback) Completed(allocationId, filePath, filename, mimetype string, size, op int) {
}
func (sc *StatusCallback) RepairCompleted(filesRepaired int) {}
