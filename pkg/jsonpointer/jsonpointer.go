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

func unescapesegment(segment string) string {
	// As per https://tools.ietf.org/html/rfc6901#section-4, process ~1 first, then ~0
	unescapedSeg := strings.ReplaceAll(segment, "~1", "/")
	return strings.ReplaceAll(unescapedSeg, "~0", "~")
}

// ToPath converts a json-pointer to an slice of property names or indicies.
func ToPath(ptr string) []string {
	segments := strings.Split(ptr, "/")
	for i := range segments {
		segments[i] = unescapesegment(segments[i])
	}
	return segments[1:]
}

// Extract takes the data structure returned by json.Unmarshal and returns the value pointed at by pointer.
func Extract(obj interface{}, pointer string) (interface{}, error) {
	if pointer == "" {
		return obj, nil
	}
	if strings.Index(pointer, "/") != 0 {
		return nil, fmt.Errorf("non-empty pointer does not start with '/': %w", ErrInvalidPointer)
	}

	segments := ToPath(pointer)
	currentObj := obj
	for _, segment := range segments {
		if slice, ok := currentObj.([]interface{}); ok {
			if index, err := strconv.Atoi(segment); err == nil {
				currentObj = slice[index]
			} else {
				return nil, ErrDoesNotExist
			}
		} else if mapObj, ok := currentObj.(map[string]interface{}); ok {
			currentObj, ok = mapObj[segment]
			if !ok {
				return nil, ErrDoesNotExist
			}
		}

	}
	return currentObj, nil
}

// ExtractParent takes the data structure returned by json.Unmarshal and returns the parent object and the key/index for the final value
func ExtractParent(obj interface{}, pointer string) (interface{}, string, error) {
	if pointer == "" {
		return nil, "", fmt.Errorf("can't get parent of top level object: %w", ErrDoesNotExist)
	}
	if strings.Index(pointer, "/") != 0 {
		return nil, "", fmt.Errorf("non-empty pointer does not start with '/': %w", ErrInvalidPointer)
	}

	parentObj, err := Extract(obj, pointer[:strings.LastIndex(pointer, "/")])
	if err != nil {
		return nil, "", err
	}
	segments := ToPath(pointer)
	return parentObj, segments[len(segments)-1], nil
}

// ExtractParentOfParent takes the data structure returned by json.Unmarshal and returns the parent 2 levels up in the hiearchy and the key/index for the final value
func ExtractParentOfParent(obj interface{}, pointer string) (interface{}, string, error) {
	if pointer == "" {
		return nil, "", fmt.Errorf("can't get parent of top level object: %w", ErrDoesNotExist)
	}
	if strings.Index(pointer, "/") != 0 {
		return nil, "", fmt.Errorf("non-empty pointer does not start with '/': %w", ErrInvalidPointer)
	}

	segments := ToPath(pointer)
	if len(segments) < 2 {
		return nil, "", fmt.Errorf("can't go beyond top level object: %w", ErrDoesNotExist)
	}

	parentObj, _, err := ExtractParent(obj, pointer[:strings.LastIndex(pointer, "/")])
	if err != nil {
		return nil, "", err
	}
	return parentObj, segments[len(segments)-2], nil
}
