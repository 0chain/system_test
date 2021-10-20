package cli_tests

import (
	"testing"
)

//move an object to another folder on blobbers
//
//Usage:
//  zbox move [flags]
//
//Flags:
//      --allocation string   Allocation ID
//      --commit              pass this option to commit the metadata transaction
//      --destpath string     Destination path for the object. Existing directory the object should be copied to
//  -h, --help                help for move
//      --remotepath string   Remote path of object to move
func TestFileMove(t *testing.T) {
	t.Parallel()

	t.Run("TODO", func(t *testing.T) {
		t.Parallel()

		// register wallet
		// faucet
		// new alloc
		// file upload
		// move file
		// list-all
	})

	// move to another existing directory
	// move to non-existing directory
	// move file to same directory (no change)
	// move to directory with existing file with same name
	// move with commit
	// move to non-existing bad directory
	// move non-existing file
	// bad allocation
	// allocation not owned

	// curator scenarios?
	// collaborator scenarios?
}
