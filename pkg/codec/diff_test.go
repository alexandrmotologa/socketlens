package codec

import (
	"testing"
)

func TestCompareJSON(t *testing.T) {
	oldJSON := []byte(`{"id": 1, "price": 100.5, "status": "pending", "items": ["a", "b"]}`)
	newJSON := []byte(`{"id": 1, "price": 105.0, "status": "completed", "items": ["a", "b", "c"], "extra": true}`)

	diff, err := CompareJSON(oldJSON, newJSON)
	if err != nil {
		t.Fatalf("compare error: %v", err)
	}

	if !diff.HasChanges {
		t.Fatal("expected changes detected")
	}

	kinds := make(map[DiffKind]int)
	for _, c := range diff.Changes {
		kinds[c.Kind]++
	}

	if kinds[DiffModified] < 2 { // price, status
		t.Errorf("expected at least 2 modified fields, got %d", kinds[DiffModified])
	}
	if kinds[DiffAdded] < 2 { // items[2], extra
		t.Errorf("expected at least 2 added fields, got %d", kinds[DiffAdded])
	}
}
