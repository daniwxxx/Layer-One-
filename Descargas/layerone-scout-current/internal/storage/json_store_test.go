package storage

import (
	"path/filepath"
	"testing"
)

func TestOpenJSONStore_LoadEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.json")
	store, err := NewJSONStore(path, "")
	if err != nil {
		t.Fatal(err)
	}
	db, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(db.Persons) != 0 {
		t.Fatal("se esperaba 0 personas")
	}
}
