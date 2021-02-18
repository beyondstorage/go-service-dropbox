// +build integration_test

package tests

import (
	"os"
	"testing"

	"github.com/google/uuid"

	dropbox "github.com/aos-dev/go-service-dropbox"
	ps "github.com/aos-dev/go-storage/v3/pairs"
	"github.com/aos-dev/go-storage/v3/types"
)

func setupTest(t *testing.T) types.Storager {
	t.Log("Setup test for dropbox")

	store, err := dropbox.NewStorager(
		ps.WithCredential(os.Getenv("STORAGE_DROPBOX_CREDENTIAL")),
		ps.WithWorkDir("/"+uuid.New().String()),
	)
	if err != nil {
		t.Errorf("new storager: %v", err)
	}
	return store
}
