package cli_model

import "time"

type Attributes struct {
	WhoPaysForReads int `json:"who_pays_for_reads,omitempty"`
}

type ListFileResult struct {
	Name            string     `json:"name"`
	Path            string     `json:"path"`
	Type            string     `json:"type"`
	Size            int64      `json:"size"`
	Hash            string     `json:"hash"`
	Mimetype        string     `json:"mimetype"`
	NumBlocks       int        `json:"num_blocks"`
	LookupHash      string     `json:"lookup_hash"`
	EncryptionKey   string     `json:"encryption_key"`
	ActualSize      int64      `json:"actual_size"`
	ActualNumBlocks int        `json:"actual_num_blocks"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Attribute       Attributes `json:"attributes"`
}
