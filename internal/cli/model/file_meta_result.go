package cli_model

type FileMetaResult struct {
	Name            string        `json:"Name"`
	Path            string        `json:"Path"`
	Type            string        `json:"Type"`
	Size            int64         `json:"Size"`
	ActualFileSize  int64         `json:"ActualFileSize"`
	LookupHash      string        `json:"LookupHash"`
	Hash            string        `json:"Hash"`
	MimeType        string        `json:"MimeType"`
	ActualNumBlocks int           `json:"ActualNumBlocks"`
	EncryptedKey    string        `json:"EncryptedKey"`
	CommitMetaTxns  []interface{} `json:"CommitMetaTxns"`
	Collaborators   []interface{} `json:"Collaborators"`
	Attribute       Attributes    `json:"attributes"`
}
