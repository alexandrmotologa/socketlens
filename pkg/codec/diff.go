package codec

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// DiffKind represents the category of difference.
type DiffKind string

const (
	DiffAdded    DiffKind = "added"
	DiffRemoved  DiffKind = "removed"
	DiffModified DiffKind = "modified"
)

// DiffChange describes a single structural or value change.
type DiffChange struct {
	Path     string      `json:"path"`
	Kind     DiffKind    `json:"kind"`
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value,omitempty"`
}

// DiffResult holds all changes between two JSON structures.
type DiffResult struct {
	HasChanges bool         `json:"has_changes"`
	Changes    []DiffChange `json:"changes"`
}

// CompareJSON takes two JSON byte slices and computes their semantic diff.
func CompareJSON(oldJSON, newJSON []byte) (*DiffResult, error) {
	if len(oldJSON) == 0 && len(newJSON) == 0 {
		return &DiffResult{HasChanges: false}, nil
	}

	var oldVal, newVal interface{}
	if len(oldJSON) > 0 {
		if err := json.Unmarshal(oldJSON, &oldVal); err != nil {
			return nil, fmt.Errorf("parsing old JSON: %w", err)
		}
	}
	if len(newJSON) > 0 {
		if err := json.Unmarshal(newJSON, &newVal); err != nil {
			return nil, fmt.Errorf("parsing new JSON: %w", err)
		}
	}

	changes := compareValues("", oldVal, newVal)
	return &DiffResult{
		HasChanges: len(changes) > 0,
		Changes:    changes,
	}, nil
}

func compareValues(path string, oldVal, newVal interface{}) []DiffChange {
	var changes []DiffChange

	if oldVal == nil && newVal != nil {
		return []DiffChange{{Path: pathOrRoot(path), Kind: DiffAdded, NewValue: newVal}}
	}
	if oldVal != nil && newVal == nil {
		return []DiffChange{{Path: pathOrRoot(path), Kind: DiffRemoved, OldValue: oldVal}}
	}

	// Compare Maps
	oldMap, oldIsMap := oldVal.(map[string]interface{})
	newMap, newIsMap := newVal.(map[string]interface{})

	if oldIsMap && newIsMap {
		allKeys := make(map[string]struct{})
		for k := range oldMap {
			allKeys[k] = struct{}{}
		}
		for k := range newMap {
			allKeys[k] = struct{}{}
		}

		for k := range allKeys {
			subPath := k
			if path != "" {
				subPath = fmt.Sprintf("%s.%s", path, k)
			}
			ov, oExists := oldMap[k]
			nv, nExists := newMap[k]

			if !oExists {
				changes = append(changes, DiffChange{Path: subPath, Kind: DiffAdded, NewValue: nv})
			} else if !nExists {
				changes = append(changes, DiffChange{Path: subPath, Kind: DiffRemoved, OldValue: ov})
			} else {
				changes = append(changes, compareValues(subPath, ov, nv)...)
			}
		}
		return changes
	}

	// Compare Slices
	oldSlice, oldIsSlice := oldVal.([]interface{})
	newSlice, newIsSlice := newVal.([]interface{})

	if oldIsSlice && newIsSlice {
		maxLen := len(oldSlice)
		if len(newSlice) > maxLen {
			maxLen = len(newSlice)
		}

		for i := 0; i < maxLen; i++ {
			subPath := fmt.Sprintf("%s[%d]", pathOrRoot(path), i)
			if i >= len(oldSlice) {
				changes = append(changes, DiffChange{Path: subPath, Kind: DiffAdded, NewValue: newSlice[i]})
			} else if i >= len(newSlice) {
				changes = append(changes, DiffChange{Path: subPath, Kind: DiffRemoved, OldValue: oldSlice[i]})
			} else {
				changes = append(changes, compareValues(subPath, oldSlice[i], newSlice[i])...)
			}
		}
		return changes
	}

	// Primitive comparison
	if !reflect.DeepEqual(oldVal, newVal) {
		changes = append(changes, DiffChange{
			Path:     pathOrRoot(path),
			Kind:     DiffModified,
			OldValue: oldVal,
			NewValue: newVal,
		})
	}

	return changes
}

func pathOrRoot(p string) string {
	if p == "" {
		return "$"
	}
	return p
}
