package storage_test

import (
	"context"
	"testing"

	"go-starter/internal/kernel/storage"
)

func FuzzOpenSeekerPathTraversal(f *testing.F) {
	f.Add("abc.jpg")
	f.Add("../etc/passwd")
	f.Add(`..\..\windows\win.ini`)
	f.Add("")
	f.Add(".")
	f.Add("..")

	store, err := storage.NewLocalStore(f.TempDir(), storage.ImageOptions{})
	if err != nil {
		f.Fatal(err)
	}

	f.Fuzz(func(t *testing.T, key string) {
		rc, _, _, err := store.OpenSeeker(context.Background(), key)
		if err == nil {
			_ = rc.Close()
		}
	})
}
