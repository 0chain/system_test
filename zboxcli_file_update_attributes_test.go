package cli_tests

import (
	"testing"
)

// update object attributes on blobbers
//
//Usage:
//  zbox update-attributes [flags]
//
//Flags:
//      --allocation string           Allocation ID
//      --commit                      pass this option to commit the metadata transaction
//  -h, --help                        help for update-attributes
//      --remotepath string           Remote path of object to rename
//      --who-pays-for-reads string   Who pays for reads: owner or 3rd_party (default "owner")
func TestFileUpdateAttributes(t *testing.T) {
	t.Parallel()

	t.Run("TODO", func(t *testing.T) {
		t.Parallel()

		// register wallet
		// faucet
		// new alloc
		// file upload
		// update file attributes
		// list-all
	})

	// update attributes of existing file
	// update attributes of existing file (no change)
	// update attributes with commit
	// update attributes of non-existing file
	// bad allocation
	// allocation not owned

	// curator scenarios?
	// collaborator scenarios?
}
