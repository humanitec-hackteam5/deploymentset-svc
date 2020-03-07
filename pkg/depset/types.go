package depset

// Set is the actual Deployment Set
type Set struct {
	Modules map[string]map[string]interface{} `json:"modules"`
	Version int                               `json:"version"`
}

// ModuleSpec is all of the data for a module.
// Keys are expected to be Helm template paths and values are the inserted values
//type ModuleSpec map[string]interface{}

// Delta is the actual Deployment Set
type Delta struct {
	Modules ModuleDeltas `json:"modules"`
}

// ModuleDeltas groups the different operations together.
type ModuleDeltas struct {
	Add    map[string]map[string]interface{} `json:"add"`
	Remove []string                          `json:"remove"`
	Update map[string][]UpdateAction         `json:"update"`
}

// UpdateAction is a representation of the main object defined in JSON Patch specified in RFC 6902 from the IETF.
// The main difference is that we only support values of type string for now.
// Operation can be one of "add", "remove", "replace"
type UpdateAction struct {
	Operation string      `json:"op"`
	Path      string      `json:"path"`
	Value     interface{} `json:"value,omitempty"`
}
