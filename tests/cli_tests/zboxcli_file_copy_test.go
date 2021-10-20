package cli_tests

import (
	"testing"
)

// copy an object to another folder on blobbers
//
//Usage:
//  zbox copy [flags]
//
//Flags:
//      --allocation string   Allocation ID
//      --commit              pass this option to commit the metadata transaction
//      --destpath string     Destination path for the object. Existing directory the object should be copied to
//  -h, --help                help for copy
//      --remotepath string   Remote path of object to copy
func TestFileCopy(t *testing.T) {
	t.Parallel()

	t.Run("TODO", func(t *testing.T) {
		t.Parallel()

		// register wallet
		// faucet
		// new alloc
		// file upload
		// copy file
		// list-all
	})

	// copy to another existing directory
	// copy to same directory
	// copy to non-existing directory
	// copy to directory with existing file with same name
	// copy with commit
	// copy to non-existing bad directory
	// copy non-existing file
	// copy but no more allocation space
	// bad allocation
	// allocation not owned

	// curator scenarios?
	// collaborator scenarios?
}
