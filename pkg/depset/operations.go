package depset

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
)

// ErrNotSupported returned when an operation is not supported (e.g. in the UpdateAction.Operation)
var ErrNotSupported = errors.New("not supported")

// ErrNotFound returned when a field required for Delta to be applied was missing.
var ErrNotFound = errors.New("not found")

func copyModuleSpec(ms ModuleSpec) ModuleSpec {
	// for now we assume that all values are actially value type and not secretly maps or slices...
	// Maybe we should use something like: https://gist.github.com/soroushjp/0ec92102641ddfc3ad5515ca76405f4d
	out := make(ModuleSpec)
	for k := range ms {
		out[k] = ms[k]
	}
	return out
}

// Apply generates a new Deployment Set from an existsing set by applying a Deployment Delat
// Both the Set and Delte objects can be big so use pointers. (Question: is this deomatic go?)
func (inputSet Set) Apply(delta Delta) (Set, error) {
	// Note: The Set structure makes a lot of use of map
	// In Go, maps are always passed by referece, so they should not be mutated
	// For this function, we need to make sure we *never* update any map inside inputSet

	// TODO: Check for conflicts.

	set := Set{Modules: make(map[string]ModuleSpec)}

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

			switch action.Operation {
			case "add":
				set.Modules[name][action.Path] = action.Value
			case "remove":
				delete(set.Modules[name], action.Path)
			case "replace":
				// Question: Do we need replace? Maybe this is just the same as an add in practice?
				set.Modules[name][action.Path] = action.Value
			default:
				return Set{}, ErrNotSupported
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

func moduleSpecDiff(left, right ModuleSpec) []UpdateAction {
	updates := make([]UpdateAction, 0, max(len(left), len(right)))

	for rightSpec := range right {
		_, exists := left[rightSpec]
		if exists {
			// config is common to both - replace
			// only update if they are different
			if !reflect.DeepEqual(left[rightSpec], right[rightSpec]) {
				updates = append(updates, UpdateAction{
					Operation: "replace",
					Path:      rightSpec,
					Value:     left[rightSpec],
				})
			}
		} else {
			// 	only in right - should be removed
			updates = append(updates, UpdateAction{
				Operation: "remove",
				Path:      rightSpec,
			})
		}
	}

	for leftSpec := range left {
		_, exists := right[leftSpec]
		if !exists {
			// config is only in right - add
			updates = append(updates, UpdateAction{
				Operation: "add",
				Path:      leftSpec,
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
			Add:    make(map[string]ModuleSpec),
			Update: make(map[string][]UpdateAction),
		},
	}
	// Find all modules that are in rightSet but not in leftSet
	// Also deal with modules that are common to both
	for rightModuleName := range rightSet.Modules {
		_, exists := leftSet.Modules[rightModuleName]
		if exists {
			// Module is common to both
			updates := moduleSpecDiff(leftSet.Modules[rightModuleName], rightSet.Modules[rightModuleName])
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

func getModuleSpecKeysAsSortedSlice(m map[string]ModuleSpec) []string {
	a := make([]string, len(m))
	i := 0
	for k := range m {
		a[i] = k
		i++
	}
	sort.StringSlice(a).Sort()
	return a
}

func getModuleSpecAsSortedSlice(m ModuleSpec) [][2]interface{} {
	sortedKeys := getMapKeysAsSortedSlice(m)

	kvpArr := make([][2]interface{}, len(m))
	for i := range sortedKeys {
		kvpArr[i] = [2]interface{}{sortedKeys[i], m[sortedKeys[i]]}
	}
	return kvpArr
}

func getModulesAsSortedSlice(m map[string]ModuleSpec) [][2]interface{} {
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

	// sepecial case for the empty set, the hash is zero
	if len(inputSet.Modules) == 0 {
		return "0000000000000000000000000000000000000000"
	}

	arrSet := [2]interface{}{"modules", getModulesAsSortedSlice(inputSet.Modules)}

	buf, _ := json.Marshal(arrSet)
	checksum := sha1.Sum(buf)
	return hex.EncodeToString(checksum[:])
}
