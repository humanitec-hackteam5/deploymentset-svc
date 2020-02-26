package jsonpointer

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/matryer/is"
)

func TestToPointer(t *testing.T) {
	is := is.New(t)
	pointer := "/hello/world/~0tilda/with~1a slash/~01/something$"
	expectedPath := []string{"hello", "world", "~tilda", "with/a slash", "~1", "something$"}
	actualPath := ToPath(pointer)
	is.Equal(expectedPath, actualPath)
}

func TestExtract(t *testing.T) {
	is := is.New(t)
	// From RFC: https://tools.ietf.org/html/rfc6901#section-5
	jsonObj := `{
      "foo":["bar", "baz"],
      "": 0,
      "a/b":1,
      "c%d":2,
      "e^f":3,
      "g|h":4,
      "i\\j":5,
      "k\"l":6,
      " ":7,
      "m~n":8
   }`
	var obj interface{}
	err := json.Unmarshal(([]byte)(jsonObj), &obj)
	is.NoErr(err)

	doTest := func(pointer string, expectedJSON string) {
		actualObj, err := Extract(obj, pointer)
		is.NoErr(err)
		actualJSON, _ := json.Marshal(actualObj)
		is.Equal(expectedJSON, string(actualJSON))
	}

	actualObj, err := Extract(obj, "")
	is.NoErr(err)
	is.Equal(obj, actualObj)

	doTest("/foo", `["bar","baz"]`)
	doTest("/foo/0", `"bar"`)
	doTest("/", `0`)
	doTest("/a~1b", `1`)
	doTest("/c%d", `2`)
	doTest("/e^f", `3`)
	doTest("/g|h", `4`)
	doTest("/i\\j", `5`)
	doTest("/k\"l", `6`)
	doTest("/ ", `7`)
	doTest("/m~0n", `8`)

}

func TestExtract_InvalidPointer(t *testing.T) {
	is := is.New(t)
	_, err := Extract(nil, "hello")
	is.True(errors.Is(err, ErrInvalidPointer))
}

func TestExtractParent(t *testing.T) {
	is := is.New(t)
	jsonObj := `{
  "array": [
    "foo",
    "bar",
    "baz"
  ],
  "object": {
    "foo": 0
  }
}`
	var obj interface{}
	err := json.Unmarshal(([]byte)(jsonObj), &obj)
	is.NoErr(err)

	doTest := func(pointer string, expectedJSON, expectedKey string) {
		actualObj, actualKey, err := ExtractParent(obj, pointer)
		is.NoErr(err)
		actualJSON, _ := json.Marshal(actualObj)
		is.Equal(expectedJSON, string(actualJSON))
		is.Equal(expectedKey, string(actualKey))
	}

	doTest("/array/0", `["foo","bar","baz"]`, `0`)
	doTest("/array/-", `["foo","bar","baz"]`, `-`)
	doTest("/object/foo", `{"foo":0}`, `foo`)
}
