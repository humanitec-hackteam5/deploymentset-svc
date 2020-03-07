package depset

import (
	"log"
	"reflect"
	"testing"
)

func orderInvariantEqual(aIn, bIn interface{}) bool {
	aType := reflect.ValueOf(aIn)
	bType := reflect.ValueOf(bIn)
	if aType.Kind() != reflect.Slice || bType.Kind() != reflect.Slice {
		return false
	}

	if aType.Len() != bType.Len() {
		log.Printf("%v != %v\n", aIn, bIn)
		return false
	}

	matchA := make([]int, aType.Len())
	matchB := make([]int, aType.Len())

	for i := 0; i < aType.Len(); i++ {
		for j := 0; j < aType.Len(); j++ {
			// Count number of duplicates in a
			if reflect.DeepEqual(aType.Index(i).Interface(), aType.Index(j).Interface()) {
				matchA[i]++
			}
			// Count number of times a[i] matches in b
			if reflect.DeepEqual(aType.Index(i).Interface(), bType.Index(j).Interface()) {
				matchB[i]++
			}
		}
	}

	for i := range matchA {
		if matchA[i] != matchB[i] {
			return false
		}
	}

	return true
}

func validateApply(inputSet Set, delta Delta, expectedSet Set, t *testing.T) {
	generatedSet, err := inputSet.Apply(delta)
	if err != nil {
		t.Errorf("Expected no error, got error: %v", err)
	} else if !reflect.DeepEqual(generatedSet, expectedSet) {
		t.Errorf("Expected: `%+v`, got `%+v`", expectedSet, generatedSet)
	}
}

func TestApplyToEmptySet(t *testing.T) {
	emptySet := Set{}
	delta := Delta{
		Modules: ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"test-module": map[string]interface{}{
					"version": "TEST_VERSION",
				},
			},
		},
	}
	expectedSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"version": "TEST_VERSION",
			},
		},
	}
	validateApply(emptySet, delta, expectedSet, t)
}

func TestApplyRemoveSingleModule(t *testing.T) {
	inputSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"version": "TEST_VERSION",
			},
		},
	}
	delta := Delta{
		Modules: ModuleDeltas{
			Remove: []string{
				"test-module",
			},
		},
	}
	expectedSet := Set{Modules: make(map[string]map[string]interface{})}
	validateApply(inputSet, delta, expectedSet, t)
}

func TestApplyRemoveModule(t *testing.T) {
	inputSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module1": map[string]interface{}{
				"version": "TEST_VERSION",
			},
			"test-module2": map[string]interface{}{
				"version": "TEST_VERSION",
			},
		},
	}
	delta := Delta{
		Modules: ModuleDeltas{
			Remove: []string{
				"test-module1",
			},
		},
	}

	expectedSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module2": map[string]interface{}{
				"version": "TEST_VERSION",
			},
		},
	}
	validateApply(inputSet, delta, expectedSet, t)
}

func TestApplyUpdateModuleAddField(t *testing.T) {
	inputSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"version": "TEST_VERSION",
			},
		},
	}
	delta := Delta{
		Modules: ModuleDeltas{
			Update: map[string][]UpdateAction{
				"test-module": []UpdateAction{
					UpdateAction{
						Operation: "add",
						Path:      "/NEW_FIELD",
						Value:     "NEW_VALUE",
					},
				},
			},
		},
	}

	expectedSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"version":   "TEST_VERSION",
				"NEW_FIELD": "NEW_VALUE",
			},
		},
	}

	validateApply(inputSet, delta, expectedSet, t)
}

func TestApplyUpdateModuleManipulateSubObject(t *testing.T) {
	inputSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"version": "TEST_VERSION",
				"configmap": map[string]interface{}{
					"HELLO":        "World!",
					"UNWANTED_KEY": "Unwanted Value!",
					"KEY":          "Value",
				},
			},
		},
	}
	delta := Delta{
		Modules: ModuleDeltas{
			Update: map[string][]UpdateAction{
				"test-module": []UpdateAction{
					UpdateAction{
						Operation: "add",
						Path:      "/configmap/NEW_KEY",
						Value:     "New Value!",
					},
					UpdateAction{
						Operation: "remove",
						Path:      "/configmap/UNWANTED_KEY",
					},
					UpdateAction{
						Operation: "replace",
						Path:      "/configmap/HELLO",
						Value:     "Alice!",
					},
				},
			},
		},
	}

	expectedSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"version": "TEST_VERSION",
				"configmap": map[string]interface{}{
					"HELLO":   "Alice!",
					"NEW_KEY": "New Value!",
					"KEY":     "Value",
				},
			},
		},
	}

	validateApply(inputSet, delta, expectedSet, t)
}

func TestApplyUpdateModuleManipulateArrayValues(t *testing.T) {
	inputSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"version": "TEST_VERSION",
				"array": []interface{}{
					"value-one",
					"value-two",
					"value-three",
					"value-four",
					"value-five",
				},
			},
		},
	}
	delta := Delta{
		Modules: ModuleDeltas{
			Update: map[string][]UpdateAction{
				"test-module": []UpdateAction{
					UpdateAction{
						Operation: "replace",
						Path:      "/array/1",
						Value:     "NEW VALUE!",
					},
					UpdateAction{
						Operation: "add",
						Path:      "/array/-",
						Value:     "LAST VALUE!",
					},
					UpdateAction{
						Operation: "add",
						Path:      "/array/3",
						Value:     "INSERTED VALUE",
					},
					UpdateAction{
						Operation: "remove",
						Path:      "/array/4",
					},
				},
			},
		},
	}

	expectedSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"version": "TEST_VERSION",
				"array": []interface{}{
					"value-one",
					"NEW VALUE!",
					"value-three",
					"INSERTED VALUE",
					"value-five",
					"LAST VALUE!",
				},
			},
		},
	}

	validateApply(inputSet, delta, expectedSet, t)
}

func TestApplyUpdateModuleManipulateArrayValuesWithOuterArray(t *testing.T) {
	inputSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"version": "TEST_VERSION",
				"array": []interface{}{
					"outer-value",
					[]interface{}{
						"value-one",
						"value-two",
						"value-three",
						"value-four",
						"value-five",
					},
				},
			},
		},
	}
	delta := Delta{
		Modules: ModuleDeltas{
			Update: map[string][]UpdateAction{
				"test-module": []UpdateAction{
					UpdateAction{
						Operation: "replace",
						Path:      "/array/1/1",
						Value:     "NEW VALUE!",
					},
					UpdateAction{
						Operation: "add",
						Path:      "/array/1/-",
						Value:     "LAST VALUE!",
					},
					UpdateAction{
						Operation: "add",
						Path:      "/array/1/3",
						Value:     "INSERTED VALUE",
					},
					UpdateAction{
						Operation: "remove",
						Path:      "/array/1/4",
					},
				},
			},
		},
	}

	expectedSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"version": "TEST_VERSION",
				"array": []interface{}{
					"outer-value",
					[]interface{}{
						"value-one",
						"NEW VALUE!",
						"value-three",
						"INSERTED VALUE",
						"value-five",
						"LAST VALUE!",
					},
				},
			},
		},
	}

	validateApply(inputSet, delta, expectedSet, t)
}

func TestApplyUpdateModuleRemoveField(t *testing.T) {
	inputSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"param01": "VALUE01",
				"param02": "VALUE02",
				"param03": "VALUE03",
			},
		},
	}
	delta := Delta{
		Modules: ModuleDeltas{
			Update: map[string][]UpdateAction{
				"test-module": []UpdateAction{
					UpdateAction{
						Operation: "remove",
						Path:      "/param02",
					},
				},
			},
		},
	}

	expectedSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"param01": "VALUE01",
				"param03": "VALUE03",
			},
		},
	}

	validateApply(inputSet, delta, expectedSet, t)
}

func TestApplyUpdateModuleReplaceField(t *testing.T) {
	inputSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"param01": "VALUE01",
				"param02": "VALUE02",
				"param03": "VALUE03",
			},
		},
	}
	delta := Delta{
		Modules: ModuleDeltas{
			Update: map[string][]UpdateAction{
				"test-module": []UpdateAction{
					UpdateAction{
						Operation: "replace",
						Path:      "/param02",
						Value:     "NEW_VALUE02",
					},
				},
			},
		},
	}

	expectedSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"param01": "VALUE01",
				"param02": "NEW_VALUE02",
				"param03": "VALUE03",
			},
		},
	}

	validateApply(inputSet, delta, expectedSet, t)
}

func TestApplyUpdateModuleNotFound(t *testing.T) {
	inputSet := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"param01": "VALUE01",
				"param02": "VALUE02",
				"param03": "VALUE03",
			},
		},
	}
	delta := Delta{
		Modules: ModuleDeltas{
			Update: map[string][]UpdateAction{
				"other-module": []UpdateAction{
					UpdateAction{
						Operation: "add",
						Path:      "/newParam04",
						Value:     "NEW_VALUE04",
					},
				},
			},
		},
	}

	_, err := inputSet.Apply(delta)
	if err != ErrNotFound {
		t.Errorf("Expected error `%v`, got `%v`", ErrNotFound, err)
	}
}

func validateDiff(left, right Set, expected Delta, t *testing.T) {
	validateDelta(left.Diff(right), expected, t)
}

func validateDelta(actual, expected Delta, t *testing.T) {
	// NOTE, we are using reflect.DeepEqual here. There are a few gotchas:
	// 1. it compares typed nils differently. e.g. nil != map[string]interface{}

	if !reflect.DeepEqual(actual.Modules.Add, expected.Modules.Add) {
		t.Errorf("Expected: `%v`, got `%v`", actual, expected)
		return
	}
	if !orderInvariantEqual(actual.Modules.Remove, expected.Modules.Remove) {
		t.Errorf("Expected: `%v`, got `%v`", actual, expected)
		return
	}
	if len(actual.Modules.Update) != len(expected.Modules.Update) {
		t.Errorf("Expected: `%v`, got `%v`", actual, expected)
		return
	}
	for module := range actual.Modules.Update {
		if _, ok := expected.Modules.Update[module]; !ok {
			t.Errorf("Expected: `%v`, got `%v`", actual, expected)
			return
		}
		if !orderInvariantEqual(actual.Modules.Update[module], expected.Modules.Update[module]) {
			t.Errorf("Expected: `%v`, got `%v`", actual, expected)
			return
		}
	}
}

func TestDiffToEmptySet(t *testing.T) {
	emptySet := Set{}
	left := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"version": "TEST_VERSION",
			},
		},
	}
	expected := Delta{
		Modules: ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"test-module": map[string]interface{}{
					"version": "TEST_VERSION",
				},
			},
			Remove: []string{},
			Update: map[string][]UpdateAction{},
		},
	}
	validateDiff(left, emptySet, expected, t)
}

func TestDiffFromEmptySet(t *testing.T) {
	emptySet := Set{}
	right := Set{
		Modules: map[string]map[string]interface{}{
			"test-module": map[string]interface{}{
				"version": "TEST_VERSION",
			},
		},
	}
	expected := Delta{
		Modules: ModuleDeltas{
			Add:    map[string]map[string]interface{}{},
			Remove: []string{"test-module"},
			Update: map[string][]UpdateAction{},
		},
	}
	validateDiff(emptySet, right, expected, t)
}

func TestDiffAllChange(t *testing.T) {
	left := Set{
		Modules: map[string]map[string]interface{}{
			"only-left": map[string]interface{}{
				"version": "TEST_VERSION_LEFT",
			},
			"in-both": map[string]interface{}{
				"only-left":  "TEST_VERSION_LEFT",
				"in-both-01": "LEFT_VALUE",
				"in-both-02": "SAME_VALUE",
			},
		},
	}
	right := Set{
		Modules: map[string]map[string]interface{}{
			"only-right": map[string]interface{}{
				"version": "TEST_VERSION_RIGHT",
			},
			"in-both": map[string]interface{}{
				"only-right": "TEST_VERSION_RIGHT",
				"in-both-01": "RIGHT_VALUE",
				"in-both-02": "SAME_VALUE",
			},
		},
	}
	expected := Delta{
		Modules: ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"only-left": map[string]interface{}{
					"version": "TEST_VERSION_LEFT",
				},
			},
			Remove: []string{"only-right"},
			Update: map[string][]UpdateAction{
				"in-both": []UpdateAction{
					UpdateAction{Operation: "remove", Path: "/only-right"},
					UpdateAction{Operation: "replace", Path: "/in-both-01", Value: "LEFT_VALUE"},
					UpdateAction{Operation: "add", Path: "/only-left", Value: "TEST_VERSION_LEFT"},
				},
			},
		},
	}
	validateDiff(left, right, expected, t)
}

func TestMargeDeltas(t *testing.T) {
	deltaA := Delta{
		Modules: ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"module-add-a": map[string]interface{}{
					"version": "VERSION_LEFT_MODULE_ADD_A",
				},
			},
			Remove: []string{"module-remove-a"},
			Update: map[string][]UpdateAction{
				"module-update-a": []UpdateAction{
					UpdateAction{Operation: "remove", Path: "/module-a/remove"},
					UpdateAction{Operation: "replace", Path: "/module-a/replace", Value: "MODULE_A_REPLACE"},
					UpdateAction{Operation: "add", Path: "/module-a/add", Value: "MODULE_A_ADD"},
				},
			},
		},
	}
	deltaB := Delta{
		Modules: ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"module-add-b": map[string]interface{}{
					"version": "VERSION_LEFT_MODULE_ADD_B",
				},
			},
			Remove: []string{"module-add-a"},
			Update: map[string][]UpdateAction{
				"module-update-a": []UpdateAction{
					UpdateAction{Operation: "remove", Path: "/module-b/remove"},
					UpdateAction{Operation: "replace", Path: "/module-b/replace", Value: "MODULE_B_REPLACE"},
					UpdateAction{Operation: "add", Path: "/module-b/add", Value: "MODULE_B_ADD"},
				},
			},
		},
	}
	expectedDelta := Delta{
		Modules: ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"module-add-b": map[string]interface{}{
					"version": "VERSION_LEFT_MODULE_ADD_B",
				},
			},
			Remove: []string{"module-remove-a", "module-add-a"},
			Update: map[string][]UpdateAction{
				"module-update-a": []UpdateAction{
					UpdateAction{Operation: "remove", Path: "/module-a/remove"},
					UpdateAction{Operation: "replace", Path: "/module-a/replace", Value: "MODULE_A_REPLACE"},
					UpdateAction{Operation: "add", Path: "/module-a/add", Value: "MODULE_A_ADD"},
					UpdateAction{Operation: "remove", Path: "/module-b/remove"},
					UpdateAction{Operation: "replace", Path: "/module-b/replace", Value: "MODULE_B_REPLACE"},
					UpdateAction{Operation: "add", Path: "/module-b/add", Value: "MODULE_B_ADD"},
				},
			},
		},
	}
	mergedDelta, err := MergeDeltas(deltaA, deltaB)
	if err != nil {
		t.Errorf("MergeDeltas returned unexpected error: %v", err)
	}
	validateDelta(mergedDelta, expectedDelta, t)
}

func TestMargeDeltas_UpdatesUpdatePreviouslyAdded(t *testing.T) {
	deltaA := Delta{
		Modules: ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"module-add-a": map[string]interface{}{
					"property01": "VALUE_01",
					"property02": "VALUE_02",
				},
			},
		},
	}
	deltaB := Delta{
		Modules: ModuleDeltas{
			Update: map[string][]UpdateAction{
				"module-add-a": []UpdateAction{
					UpdateAction{Operation: "remove", Path: "/property01"},
					UpdateAction{Operation: "replace", Path: "/property02", Value: "REPLACED_VALUE"},
					UpdateAction{Operation: "add", Path: "/newProperty", Value: "NEW_VALUE"},
				},
			},
		},
	}
	expectedDelta := Delta{
		Modules: ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"module-add-a": map[string]interface{}{
					"property02":  "REPLACED_VALUE",
					"newProperty": "NEW_VALUE",
				},
			},
		},
	}
	mergedDelta, err := MergeDeltas(deltaA, deltaB)
	if err != nil {
		t.Errorf("MergeDeltas returned unexpected error: %v", err)
	}
	validateDelta(mergedDelta, expectedDelta, t)
}
func TestMargeDeltas_UpdatesAdded(t *testing.T) {
	deltaA := Delta{
		Modules: ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"module-add-a": map[string]interface{}{
					"property01": "VALUE_01",
					"property02": "VALUE_02",
				},
			},
		},
	}
	deltaB := Delta{
		Modules: ModuleDeltas{
			Update: map[string][]UpdateAction{
				"module-add-b": []UpdateAction{
					UpdateAction{Operation: "remove", Path: "/property01"},
					UpdateAction{Operation: "replace", Path: "/property02", Value: "REPLACED_VALUE"},
					UpdateAction{Operation: "add", Path: "/newProperty", Value: "NEW_VALUE"},
				},
			},
		},
	}
	expectedDelta := Delta{
		Modules: ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"module-add-a": map[string]interface{}{
					"property01": "VALUE_01",
					"property02": "VALUE_02",
				},
			},
			Update: map[string][]UpdateAction{
				"module-add-b": []UpdateAction{
					UpdateAction{Operation: "remove", Path: "/property01"},
					UpdateAction{Operation: "replace", Path: "/property02", Value: "REPLACED_VALUE"},
					UpdateAction{Operation: "add", Path: "/newProperty", Value: "NEW_VALUE"},
				},
			},
		},
	}
	mergedDelta, err := MergeDeltas(deltaA, deltaB)
	if err != nil {
		t.Errorf("MergeDeltas returned unexpected error: %v", err)
	}
	validateDelta(mergedDelta, expectedDelta, t)
}

func TestMargeDeltas_DuplicateRemovesAddOverride(t *testing.T) {
	deltaA := Delta{
		Modules: ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"module-add-a": map[string]interface{}{
					"version": "VERSION_LEFT_MODULE_ADD_A",
				},
			},
			Remove: []string{"module-remove-a"},
			Update: map[string][]UpdateAction{},
		},
	}
	deltaB := Delta{
		Modules: ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"module-add-a": map[string]interface{}{
					"version": "VERSION_LEFT_MODULE_ADD_B",
				},
			},
			Remove: []string{"module-remove-a"},
			Update: map[string][]UpdateAction{},
		},
	}
	expectedDelta := Delta{
		Modules: ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"module-add-a": map[string]interface{}{
					"version": "VERSION_LEFT_MODULE_ADD_B",
				},
			},
			Remove: []string{"module-remove-a"},
			Update: map[string][]UpdateAction{},
		},
	}
	mergedDelta, err := MergeDeltas(deltaA, deltaB)
	if err != nil {
		t.Errorf("MergeDeltas returned unexpected error: %v", err)
	}
	validateDelta(mergedDelta, expectedDelta, t)
}

func TestMargeDeltas_RemoveRemove(t *testing.T) {
	deltaA := Delta{
		Modules: ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"module-add-a": map[string]interface{}{
					"version": "VERSION_LEFT_MODULE_ADD_A",
				},
			},
			Remove: []string{"module-remove-to-remove", "module-remove-a"},
			Update: map[string][]UpdateAction{},
		},
	}
	deltaB := Delta{
		Modules: ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"module-remove-to-remove": nil,
			},
			Remove: []string{},
			Update: map[string][]UpdateAction{},
		},
	}
	expectedDelta := Delta{
		Modules: ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"module-add-a": map[string]interface{}{
					"version": "VERSION_LEFT_MODULE_ADD_A",
				},
			},
			Remove: []string{"module-remove-a"},
			Update: map[string][]UpdateAction{},
		},
	}
	mergedDelta, err := MergeDeltas(deltaA, deltaB)
	if err != nil {
		t.Errorf("MergeDeltas returned unexpected error: %v", err)
	}
	validateDelta(mergedDelta, expectedDelta, t)
}

func validateHash(inputSet Set, expected string, t *testing.T) {
	actual := inputSet.Hash()
	if expected != actual {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}

func TestHashEmptgySetNil(t *testing.T) {
	inputSet := Set{}
	expectedHash := "0000000000000000000000000000000000000000"

	validateHash(inputSet, expectedHash, t)
}

// This test is mainly to ensure hashes do not change unexpectadly
func TestHashGeneralCase(t *testing.T) {
	inputSet := Set{
		Modules: map[string]map[string]interface{}{
			"first-module": map[string]interface{}{
				"StringParam": "Some string!",
				"IntParam":    123,
				"FloatParam":  125.5,
				"BoolParam":   true,
			},
			"another-one": map[string]interface{}{
				"version": "TEST_VERSION",
				"param":   "TEST_param",
			},
		},
	}
	expectedHash := "b9fe8e88cfee30dff184e3bd8985421942db2e54233ea736cbac07c9fcba7814"

	validateHash(inputSet, expectedHash, t)
}
