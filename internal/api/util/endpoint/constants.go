package endpoint

// Statuses of http based responses
const (
	HttpOkStatus       = "200 OK"
	HttpNotFoundStatus = "400 Bad Request"
)

// Addresses of SC
const (
	FaucetSmartContractAddress  = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3"
	StorageSmartContractAddress = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
)

// Statuses of transactions
const (
	TxSuccessfulStatus = iota + 1
	TxUnsuccessfulStatus
)
