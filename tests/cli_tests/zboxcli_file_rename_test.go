package cli_tests

import (
	"testing"
)

// rename an object on blobbers
//
//Usage:
//  zbox rename [flags]
//
//Flags:
//      --allocation string   Allocation ID
//      --commit              pass this option to commit the metadata transaction
//      --destname string     New Name for the object (Only the name and not the path). Include the file extension if applicable
//  -h, --help                help for rename
//      --remotepath string   Remote path of object to rename

func TestFileRename(t *testing.T) {
	t.Parallel()

	t.Run("TODO", func(t *testing.T) {
		t.Parallel()

		// register wallet
		// faucet
		// new alloc
		// file upload
		// rename file
		// list-all
	})

	// rename to new filename with extension
	// rename to new filename without extension
	// rename file to same filename (no change)
	// rename to same filename as existing
	// rename with commit
	// rename to bad filename
	// rename non-existing file
	// bad allocation
	// allocation not owned

	// curator scenarios?
	// collaborator scenarios?
}
