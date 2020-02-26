package jsonpointer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidPointer indicates that the pointer syntax is invalid
var ErrInvalidPointer = errors.New("invalid json-pointer syntax")

// ErrDoesNotExist indicates that the pointer references a value that does not exist
var ErrDoesNotExist = errors.New("value does not exist")

func unescapeSegmant(segmant string) string {
	// As per https://tools.ietf.org/html/rfc6901#section-4, process ~1 first, then ~0
	unescapedSeg := strings.ReplaceAll(segmant, "~1", "/")
	return strings.ReplaceAll(unescapedSeg, "~0", "~")
}

// ToPath converts a json-pointer to an slice of property names or indicies.
func ToPath(ptr string) []string {
	segmants := strings.Split(ptr, "/")
	for i := range segmants {
		segmants[i] = unescapeSegmant(segmants[i])
	}
	return segmants[1:]
}

// Extract takes the data structure returned by json.Unmarshal and returns the value pointed at by pointer.
func Extract(obj interface{}, pointer string) (interface{}, error) {
	if pointer == "" {
		return obj, nil
	}
	if strings.Index(pointer, "/") != 0 {
		return nil, fmt.Errorf("non-empty pointer does not start with '/': %w", ErrInvalidPointer)
	}

	segmants := ToPath(pointer)
	currentObj := obj
	for _, segmant := range segmants {
		if slice, ok := currentObj.([]interface{}); ok {
			if index, err := strconv.Atoi(segmant); err == nil {
				currentObj = slice[index]
			} else {
				return nil, ErrDoesNotExist
			}
		} else if mapObj, ok := currentObj.(map[string]interface{}); ok {
			currentObj, ok = mapObj[segmant]
			if !ok {
				return nil, ErrDoesNotExist
			}
		}

	}
	return currentObj, nil
}
