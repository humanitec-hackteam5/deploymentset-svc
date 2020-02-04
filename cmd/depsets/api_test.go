package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/matryer/is"
	"humanitec.io/deploymentset-svc/pkg/depset"
)

// Custom matcher that only looks at the set part of a SetWrapper
type justSet struct{ s depset.Set }

func JustSetEq(s depset.Set) gomock.Matcher {
	return &justSet{s}
}

func (s *justSet) Matches(x interface{}) bool {
	setToTest, ok := x.(SetWrapper)
	if !ok {
		return false
	}
	return reflect.DeepEqual(s.s, setToTest.Content)
}

func (s *justSet) String() string {
	return fmt.Sprintf("%v", s.s)
}

func ExecuteRequest(m modeler, method, url string, body *bytes.Buffer, t *testing.T) *httptest.ResponseRecorder {
	server := server{
		model: m,
	}
	server.setupRoutes()

	var req *http.Request
	var err error
	if body == nil {
		req, err = http.NewRequest(method, url, nil)
	} else {
		req, err = http.NewRequest(method, url, body)
	}
	if err != nil {
		t.Errorf("creating request: %v", err)
	}

	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	return w
}

func TestGetSet(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	setID := "0123456789ABCDEFDEADBEEFDEADBEEFDEADBEEF"
	expectedSetWrapper := SetWrapper{
		Metadata: SetMetadata{
			CreatedAt: time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
		},
		Content: depset.Set{
			Modules: map[string]depset.ModuleSpec{
				"test-module": depset.ModuleSpec{
					"version": "TEST_VERSION",
				},
			},
		},
	}

	m.
		EXPECT().
		selectSet(orgID, appID, setID).
		Return(expectedSetWrapper, nil).
		Times(1)

	res := ExecuteRequest(m, "GET", fmt.Sprintf("/orgs/%s/apps/%s/sets/%s", orgID, appID, setID), nil, t)

	is.Equal(res.Code, http.StatusOK) // Should return 200

	var returnedSetWrapper SetWrapper
	json.Unmarshal(res.Body.Bytes(), &returnedSetWrapper)

	is.Equal(returnedSetWrapper, expectedSetWrapper) // Returnned Set should match initilal set

}

func TestGetAllSets(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	expectedSetWrappers := []SetWrapper{
		SetWrapper{
			Metadata: SetMetadata{
				CreatedAt: time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
			},
			Content: depset.Set{
				Modules: map[string]depset.ModuleSpec{
					"test-module": depset.ModuleSpec{
						"version": "TEST_VERSION",
					},
				},
			},
		},
		SetWrapper{
			Metadata: SetMetadata{
				CreatedAt: time.Date(2020, time.January, 1, 2, 0, 0, 0, time.UTC),
			},
			Content: depset.Set{
				Modules: map[string]depset.ModuleSpec{
					"test-module2": depset.ModuleSpec{
						"version": "TEST_VERSION2",
					},
				},
			},
		},
	}

	m.
		EXPECT().
		selectAllSets(orgID, appID).
		Return(expectedSetWrappers, nil).
		Times(1)

	res := ExecuteRequest(m, "GET", fmt.Sprintf("/orgs/%s/apps/%s/sets", orgID, appID), nil, t)

	is.Equal(res.Code, http.StatusOK) // Should return 200

	var returnedSetWrappers []SetWrapper
	json.Unmarshal(res.Body.Bytes(), &returnedSetWrappers)

	is.Equal(returnedSetWrappers, expectedSetWrappers) // Returned Sets should match initilal sets

}

func TestGetAllSets_NoSets(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	expectedSetWrappers := []SetWrapper{}

	m.
		EXPECT().
		selectAllSets(orgID, appID).
		Return(expectedSetWrappers, nil).
		Times(1)

	res := ExecuteRequest(m, "GET", fmt.Sprintf("/orgs/%s/apps/%s/sets", orgID, appID), nil, t)

	is.Equal(res.Code, http.StatusOK) // Should return 200

	is.Equal(res.Body.String(), "[]") // Returned Sets should be empty array

}

func TestApplyDelta(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	delta := depset.Delta{
		Modules: depset.ModuleDeltas{
			Add: map[string]depset.ModuleSpec{
				"test-module02": depset.ModuleSpec{
					"version": "TEST_VERSION02",
				},
			},
		},
	}
	inputSetID := "27036a0c4ce1cda91addbd67ca65d499dfbeb9d0"
	inputSet := depset.Set{
		Modules: map[string]depset.ModuleSpec{
			"test-module01": depset.ModuleSpec{
				"version": "TEST_VERSION01",
			},
		},
	}

	expectedSet := depset.Set{
		Modules: map[string]depset.ModuleSpec{
			"test-module01": depset.ModuleSpec{
				"version": "TEST_VERSION01",
			},
			"test-module02": depset.ModuleSpec{
				"version": "TEST_VERSION02",
			},
		},
	}

	m.
		EXPECT().
		selectRawSet(gomock.Eq(orgID), gomock.Eq(appID), inputSetID).
		Return(inputSet, nil).
		Times(1)

	m.
		EXPECT().
		insertSet(gomock.Eq(orgID), gomock.Eq(appID), JustSetEq(expectedSet)).
		Return(nil).
		Times(1)

	buf, err := json.Marshal(delta)
	is.NoErr(err)
	body := bytes.NewBuffer(buf)

	res := ExecuteRequest(m, "POST", fmt.Sprintf("/orgs/%s/apps/%s/sets/%s", orgID, appID, inputSetID), body, t)

	is.Equal(res.Code, http.StatusCreated) // Should return 201

	var outputID string
	json.Unmarshal(res.Body.Bytes(), &outputID)

	is.Equal(outputID, "e1baae2d638987e1ae70ab1a1cec6d4905f780c9") // Returned Sets should match initilal sets

}

func TestApplyDelta_ToZeroSet(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	delta := depset.Delta{
		Modules: depset.ModuleDeltas{
			Add: map[string]depset.ModuleSpec{
				"test-module01": depset.ModuleSpec{
					"version": "TEST_VERSION01",
				},
			},
		},
	}
	inputSetID := "0"

	expectedSet := depset.Set{
		Modules: map[string]depset.ModuleSpec{
			"test-module01": depset.ModuleSpec{
				"version": "TEST_VERSION01",
			},
		},
	}

	m.
		EXPECT().
		insertSet(gomock.Eq(orgID), gomock.Eq(appID), JustSetEq(expectedSet)).
		Return(nil).
		Times(1)

	buf, err := json.Marshal(delta)
	is.NoErr(err)
	body := bytes.NewBuffer(buf)

	res := ExecuteRequest(m, "POST", fmt.Sprintf("/orgs/%s/apps/%s/sets/%s", orgID, appID, inputSetID), body, t)

	is.Equal(res.Code, http.StatusCreated) // Should return 201 (As the new set was added)

	var outputID string
	json.Unmarshal(res.Body.Bytes(), &outputID)

	is.Equal(outputID, "4efb2d1ae4f101a1ef4e0a08705910191868c5cc")

}

func TestApplyDelta_SetAlreadyExists(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	delta := depset.Delta{
		Modules: depset.ModuleDeltas{
			Add: map[string]depset.ModuleSpec{
				"test-module01": depset.ModuleSpec{
					"version": "TEST_VERSION01",
				},
			},
		},
	}
	inputSetID := "0"

	expectedSet := depset.Set{
		Modules: map[string]depset.ModuleSpec{
			"test-module01": depset.ModuleSpec{
				"version": "TEST_VERSION01",
			},
		},
	}

	m.
		EXPECT().
		insertSet(gomock.Eq(orgID), gomock.Eq(appID), JustSetEq(expectedSet)).
		Return(ErrAlreadyExists).
		Times(1)

	buf, err := json.Marshal(delta)
	is.NoErr(err)
	body := bytes.NewBuffer(buf)

	res := ExecuteRequest(m, "POST", fmt.Sprintf("/orgs/%s/apps/%s/sets/%s", orgID, appID, inputSetID), body, t)

	is.Equal(res.Code, http.StatusOK) // Should return 200

	var outputID string
	json.Unmarshal(res.Body.Bytes(), &outputID)

	is.Equal(outputID, "4efb2d1ae4f101a1ef4e0a08705910191868c5cc")

}

func TestApplyDelta_InputSetIdUnknown(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	delta := depset.Delta{
		Modules: depset.ModuleDeltas{
			Add: map[string]depset.ModuleSpec{
				"test-module01": depset.ModuleSpec{
					"version": "TEST_VERSION01",
				},
			},
		},
	}
	inputSetID := "4efb2d1ae4f101a1ef4e0a08705910191868c5cc"

	m.
		EXPECT().
		selectRawSet(orgID, appID, inputSetID).
		Return(depset.Set{}, ErrNotFound).
		Times(1)

	buf, err := json.Marshal(delta)
	is.NoErr(err)
	body := bytes.NewBuffer(buf)

	res := ExecuteRequest(m, "POST", fmt.Sprintf("/orgs/%s/apps/%s/sets/%s", orgID, appID, inputSetID), body, t)

	is.Equal(res.Code, http.StatusNotFound) // Should return 404
}

func TestApplyDelta_DeltaNotCompatibleToInputSet(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	delta := depset.Delta{
		Modules: depset.ModuleDeltas{
			Update: map[string][]depset.UpdateAction{
				"test-module": []depset.UpdateAction{
					depset.UpdateAction{
						Operation: "replace",
						Path:      "param",
						Value:     "NEW_VALUE",
					},
				},
			},
		},
	}
	inputSetID := "4efb2d1ae4f101a1ef4e0a08705910191868c5cc"
	inputSet := depset.Set{
		Modules: map[string]depset.ModuleSpec{
			"other-module": depset.ModuleSpec{
				"other-param": "TEST_VERSION01",
			},
		},
	}

	m.
		EXPECT().
		selectRawSet(orgID, appID, inputSetID).
		Return(inputSet, nil).
		Times(1)

	buf, err := json.Marshal(delta)
	is.NoErr(err)
	body := bytes.NewBuffer(buf)

	res := ExecuteRequest(m, "POST", fmt.Sprintf("/orgs/%s/apps/%s/sets/%s", orgID, appID, inputSetID), body, t)

	is.Equal(res.Code, http.StatusBadRequest) // Should return 400
}

func TestDiff_FromEmptySet(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	leftSetID := "4efb2d1ae4f101a1ef4e0a08705910191868c5cc"
	leftSet := depset.Set{
		Modules: map[string]depset.ModuleSpec{
			"test-module": depset.ModuleSpec{
				"version": "TEST_VERSION",
			},
		},
	}
	expected := depset.Delta{
		Modules: depset.ModuleDeltas{
			Add: map[string]depset.ModuleSpec{
				"test-module": depset.ModuleSpec{
					"version": "TEST_VERSION",
				},
			},
			Remove: []string{},
			Update: map[string][]depset.UpdateAction{},
		},
	}

	m.
		EXPECT().
		selectRawSet(orgID, appID, leftSetID).
		Return(leftSet, nil).
		Times(1)

	res := ExecuteRequest(m, "GET", fmt.Sprintf("/orgs/%s/apps/%s/sets/%s?diff=%s", orgID, appID, leftSetID, "0"), nil, t)

	var actualDelta depset.Delta
	json.Unmarshal(res.Body.Bytes(), &actualDelta)

	is.Equal(actualDelta, expected)
}
