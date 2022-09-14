package model

//type ExecuteForAllServiceProvidersRequest struct {
//	// URL builder used for further URL preprocessing
//	URLBuilder *URLBuilder
//
//	// Service type which execution wanted to be performed for
//	ServiceProviderType int
//
//	// Method used for all executions
//	Method int
//}

type ExecutionRequest struct {
	FormData map[string]string

	QueryParams map[string]string

	Headers map[string]string

	Body interface{}

	Dst interface{}

	RequiredStatusCode int
}

type TransactionData struct {
	Name  string      `json:"name"`
	Input interface{} `json:"input"`
}

type TransactionPutResponse struct {
	Async  bool `json:"async"`
	Entity struct {
		PublicKey         string `json:"public_key,omitempty"`
		Version           string `json:"version"`
		ClientId          string `json:"client_id"`
		ToClientId        string `json:"to_client_id"`
		TransactionData   string `json:"transaction_data"`
		TransactionValue  int64  `json:"transaction_value"`
		CreationDate      int64  `json:"creation_date"`
		TransactionFee    int64  `json:"transaction_fee"`
		TransactionType   int    `json:"transaction_type"`
		TransactionOutput string `json:"transaction_output,omitempty"`
		TxnOutputHash     string `json:"txn_output_hash"`
		TransactionNonce  int    `json:"transaction_nonce"`
		Hash              string `json:"hash"`
		ChainId           string `json:"chain_id"`
		Signature         string `json:"signature"`
		TransactionStatus int    `json:"transaction_status"`
	} `json:"entity"`
}

type TransactionPutRequest struct {
	PublicKey         string `json:"public_key,omitempty"`
	Version           string `json:"version"`
	ClientId          string `json:"client_id"`
	ToClientId        string `json:"to_client_id"`
	TransactionData   string `json:"transaction_data"`
	TransactionValue  int64  `json:"transaction_value"`
	CreationDate      int64  `json:"creation_date"`
	TransactionFee    int64  `json:"transaction_fee"`
	TransactionType   int    `json:"transaction_type"`
	TransactionOutput string `json:"transaction_output,omitempty"`
	TxnOutputHash     string `json:"txn_output_hash"`
	TransactionNonce  int    `json:"transaction_nonce"`
}

type TransactionGetConfirmationRequest struct {
	Hash string
}

type TransactionGetConfirmationResponse struct {
}

