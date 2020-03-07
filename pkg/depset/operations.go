package depset

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"

	"humanitec.io/deploymentset-svc/pkg/jsonpointer"
)

// ErrNotSupported returned when an operation is not supported (e.g. in the UpdateAction.Operation)
var ErrNotSupported = errors.New("not supported")

// ErrNotFound returned when a field required for Delta to be applied was missing.
var ErrNotFound = errors.New("not found")

// ErrTypeMismatch returned when the type of an object is not what was expected.
var ErrTypeMismatch = errors.New("type mismatch")

func copyModuleSpec(ms map[string]interface{}) map[string]interface{} {
	// for now we assume that all values are actially value type and not secretly maps or slices...
	// Maybe we should use something like: https://gist.github.com/soroushjp/0ec92102641ddfc3ad5515ca76405f4d
	out := make(map[string]interface{})
	for k := range ms {
		out[k] = ms[k]
	}
	return out
}

// applyUpdateAction applies a JSON-PATCH (https://tools.ietf.org/html/rfc6902) action to a structure derrived from a
// JSON object.
// The update happens in place. If an error is thrown, the state of object is undefined.
func applyUpdateAction(action UpdateAction, object map[string]interface{}) error {
	parent, key, err := jsonpointer.ExtractParent(object, action.Path)
	if err != nil {
		return fmt.Errorf("path `%s`: %w", action.Path, err)
	}

	if mapObj, ok := parent.(map[string]interface{}); ok {
		switch action.Operation {
		case "add":
			mapObj[key] = action.Value

		case "remove":
			delete(mapObj, key)

		case "replace":
			if _, ok := mapObj[key]; ok {
				mapObj[key] = action.Value
			} else {
				return fmt.Errorf("path `%s` does not exist: %w", action.Path, ErrNotFound)
			}

		default:
			return ErrNotSupported
		}
	} else if slice, ok := parent.([]interface{}); ok {
		// Becasue we need to manipulate the slice which might involve creating a new slice, we need the
		// parent object of the slice.
		parentOfParent, parentKey, _ := jsonpointer.ExtractParentOfParent(object, action.Path)
		switch action.Operation {
		case "add":
			var target []interface{}
			if key == "-" {
				target = append(slice, action.Value)
			} else {
				if index, err := strconv.Atoi(key); err == nil {
					if index < len(slice) {
						target = append(slice[:index], append([]interface{}{action.Value}, slice[index:]...)...)
					} else {
						return fmt.Errorf("index in path `%s` out of range: %w", action.Path, ErrNotFound)
					}
				} else {
					return fmt.Errorf("path `%s` refers to array and does not have a numerical index: %w", action.Path, ErrTypeMismatch)
				}
			}
			if parentMapObj, ok := parentOfParent.(map[string]interface{}); ok {
				parentMapObj[parentKey] = target
			} else {
				parentSlice := parentOfParent.([]interface{})
				parentIndex, _ := strconv.Atoi(parentKey) // We know this works because it has worked in ExtractParent earlier
				parentSlice[parentIndex] = target
			}

		case "remove":
			if index, err := strconv.Atoi(key); err == nil {
				if index < len(slice) {
					if parentMapObj, ok := parentOfParent.(map[string]interface{}); ok {
						parentMapObj[parentKey] = append(slice[:index], slice[index+1:]...)
					} else {
						parentSlice := parentOfParent.([]interface{})
						parentIndex, _ := strconv.Atoi(parentKey) // We know this works because it has worked in ExtractParent earlier
						parentSlice[parentIndex] = append(slice[:index], slice[index+1:]...)
					}
				} else {
					return fmt.Errorf("index in path `%s` out of range: %w", action.Path, ErrNotFound)
				}
			} else {
				return fmt.Errorf("path `%s` refers to array and does not have a numerical index: %w", action.Path, ErrTypeMismatch)
			}
		case "replace":
			if index, err := strconv.Atoi(key); err == nil {
				if index < len(slice) {
					slice[index] = action.Value
				} else {
					return fmt.Errorf("index in path `%s` out of range: %w", action.Path, ErrNotFound)
				}
			} else {
				return fmt.Errorf("path `%s` refers to array and does not have a numerical index: %w", action.Path, ErrTypeMismatch)
			}

		default:
			return ErrNotSupported
		}
	} else {
		return fmt.Errorf("parent of path `%s` must be an array or object to be updateable. got (%v): %w", action.Path, reflect.TypeOf(parent), ErrTypeMismatch)
	}
	return nil
}

// Apply generates a new Deployment Set from an existsing set by applying a Deployment Delta.
func (inputSet Set) Apply(delta Delta) (Set, error) {
	// Note: The Set structure makes a lot of use of map
	// In Go, maps are always passed by referece, so they should not be mutated
	// For this function, we need to make sure we *never* update any map inside inputSet

	// TODO: Check for conflicts.

	set := Set{
		Modules: make(map[string]map[string]interface{}),
		Version: 0,
	}

	removeModules := make(map[string]bool)
	for _, name := range delta.Modules.Remove {
		// Question: Should we check if a module to be removes actually exists?
		// Probably not, as the "desired state" would be no module.
		removeModules[name] = true
	}

	// Remove modules
	// The code actually adds everything that is not marked as remove
	for name := range inputSet.Modules {
		if !removeModules[name] {
			set.Modules[name] = copyModuleSpec(inputSet.Modules[name])
		}
	}

	// Add modules
	for name, values := range delta.Modules.Add {
		// Question: Should we check if module exists in set *before* adding it. Otherwise an add becomes a replace.
		// Probably not, as the "desired state" would be this module

		set.Modules[name] = copyModuleSpec(values)
	}

	// Update Modules
	for name, values := range delta.Modules.Update {
		if _, ok := set.Modules[name]; !ok {
			return Set{}, ErrNotFound
		}
		// Note, that we already made a copy of the map in the "remove" section
		for _, action := range values {
			err := applyUpdateAction(action, set.Modules[name])
			if err != nil {
				return Set{}, fmt.Errorf("module %s: %w", name, err)
			}
		}
	}

	return set, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func ModuleSpecDiff(left, right map[string]interface{}) []UpdateAction {
	updates := make([]UpdateAction, 0, max(len(left), len(right)))

	for rightSpec := range right {
		_, exists := left[rightSpec]
		if exists {
			// config is common to both - replace
			// only update if they are different
			if !reflect.DeepEqual(left[rightSpec], right[rightSpec]) {
				updates = append(updates, UpdateAction{
					Operation: "replace",
					Path:      "/" + rightSpec,
					Value:     left[rightSpec],
				})
			}
		} else {
			// 	only in right - should be removed
			updates = append(updates, UpdateAction{
				Operation: "remove",
				Path:      "/" + rightSpec,
			})
		}
	}

	for leftSpec := range left {
		_, exists := right[leftSpec]
		if !exists {
			// config is only in right - add
			updates = append(updates, UpdateAction{
				Operation: "add",
				Path:      "/" + leftSpec,
				Value:     left[leftSpec],
			})
		}
	}
	return updates
}

// Diff generates the Delta between two sets. Specifically, if the generated delta is applied to rightSet, leftSet is
// generated.
func (leftSet Set) Diff(rightSet Set) Delta {
	delta := Delta{
		Modules: ModuleDeltas{
			Remove: make([]string, 0, max(len(leftSet.Modules), len(rightSet.Modules))),
			Add:    make(map[string]map[string]interface{}),
			Update: make(map[string][]UpdateAction),
		},
	}
	// Find all modules that are in rightSet but not in leftSet
	// Also deal with modules that are common to both
	for rightModuleName := range rightSet.Modules {
		_, exists := leftSet.Modules[rightModuleName]
		if exists {
			// Module is common to both
			updates := ModuleSpecDiff(leftSet.Modules[rightModuleName], rightSet.Modules[rightModuleName])
			if len(updates) > 0 {
				delta.Modules.Update[rightModuleName] = updates
			}
		} else {
			// Module is only in right - mark to remove
			delta.Modules.Remove = append(delta.Modules.Remove, rightModuleName)
		}
	}

	// Find all modules that are in leftSet but not in leftSet
	for leftModuleName := range leftSet.Modules {
		_, exists := rightSet.Modules[leftModuleName]
		if !exists {
			// Module is only in left - add it
			delta.Modules.Add[leftModuleName] = leftSet.Modules[leftModuleName]
		}
	}
	return delta
}

func removeDuplicates(a []string) []string {
	buf := make(map[string]int)
	b := make([]string, 0, len(buf))
	for i := range a {
		buf[a[i]] = buf[a[i]] + 1
	}

	for i := range a {
		buf[a[i]]--
		if 0 == buf[a[i]] {
			b = append(b, a[i])
		}
	}
	return b
}

func removeFromList(a []string, value string) []string {
	b := make([]string, 0, len(a))
	for i := range a {
		if a[i] != value {
			b = append(b, a[i])
		}
	}
	return b
}

// MergeDeltas combines an array of deltas into a single delta.
// NOTE: Order matters. E.g. Update
// NOTE: This implementation updates baseDelta in place...
func MergeDeltas(baseDelta Delta, deltas ...Delta) (Delta, error) {
	// Sanitize Input, maps should not be nil:
	if nil == baseDelta.Modules.Add {
		baseDelta.Modules.Add = make(map[string]map[string]interface{})
	}
	if nil == baseDelta.Modules.Update {
		baseDelta.Modules.Update = make(map[string][]UpdateAction)
	}

	for deltaIndex, delta := range deltas {
		for _, removeModuleName := range delta.Modules.Remove {
			delete(baseDelta.Modules.Add, removeModuleName)
			delete(baseDelta.Modules.Update, removeModuleName)
		}
		baseDelta.Modules.Remove = removeDuplicates(append(baseDelta.Modules.Remove, delta.Modules.Remove...))

		for addModuleName, addModule := range delta.Modules.Add {
			if nil == delta.Modules.Add[addModuleName] {
				baseDelta.Modules.Remove = removeFromList(baseDelta.Modules.Remove, addModuleName)
			} else {
				// Override an existing add for the module, or create a new one
				baseDelta.Modules.Add[addModuleName] = addModule

				// Override any existing updates for the module
				delete(baseDelta.Modules.Update, addModuleName)
			}
		}

		for updateModuleName, moduleUpdates := range delta.Modules.Update {
			if addModule, ok := baseDelta.Modules.Add[updateModuleName]; ok {
				// The module has been added previously. Rather than appending the updates,
				// we can update the original add.
				for _, action := range moduleUpdates {
					err := applyUpdateAction(action, addModule)
					if err != nil {
						return Delta{}, fmt.Errorf("updates in module `%s` for delta at index %d not compatible with added module: %w", updateModuleName, deltaIndex, err)
					}
				}
			} else {
				// for now, just append.
				// In future we can combine updates into the smallest set of update statements...
				if _, ok := baseDelta.Modules.Update[updateModuleName]; !ok {
					baseDelta.Modules.Update[updateModuleName] = moduleUpdates
				} else {
					baseDelta.Modules.Update[updateModuleName] = append(baseDelta.Modules.Update[updateModuleName], moduleUpdates...)
				}
			}
		}
	}
	return baseDelta, nil
}

func getMapKeysAsSortedSlice(m map[string]interface{}) []string {
	a := make([]string, len(m))
	i := 0
	for k := range m {
		a[i] = k
		i++
	}
	sort.StringSlice(a).Sort()
	return a
}

func getModuleSpecKeysAsSortedSlice(m map[string]map[string]interface{}) []string {
	a := make([]string, len(m))
	i := 0
	for k := range m {
		a[i] = k
		i++
	}
	sort.StringSlice(a).Sort()
	return a
}

func getModuleSpecAsSortedSlice(m map[string]interface{}) [][2]interface{} {
	sortedKeys := getMapKeysAsSortedSlice(m)

	kvpArr := make([][2]interface{}, len(m))
	for i := range sortedKeys {
		kvpArr[i] = [2]interface{}{sortedKeys[i], m[sortedKeys[i]]}
	}
	return kvpArr
}

func getModulesAsSortedSlice(m map[string]map[string]interface{}) [][2]interface{} {
	sortedModules := getModuleSpecKeysAsSortedSlice(m)

	kvpArr := make([][2]interface{}, len(m))
	for i := range sortedModules {
		kvpArr[i] = [2]interface{}{sortedModules[i], getModuleSpecAsSortedSlice(m[sortedModules[i]])}
	}
	return kvpArr
}

// Hash generates an invarient id for a Deployment Set
func (inputSet Set) Hash() string {
	// For now, we hack it by converting the deployment set into an array structure
	// This is because we cannot control key order using the built in go json serializer.

	// sepecial case for the empty set, the hash is zero
	if len(inputSet.Modules) == 0 {
		return "0000000000000000000000000000000000000000000"
	}

	arrSet := [2]interface{}{"modules", getModulesAsSortedSlice(inputSet.Modules)}

	buf, _ := json.Marshal(arrSet)
	checksum := sha256.Sum256(buf)

	// RawURLEncoding makes for URL safe IDs that don't have trailing '='. This means no URL encoding required.
	return base64.RawURLEncoding.EncodeToString(checksum[:])
}
