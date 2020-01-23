package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/matryer/is"
	"humanitec.io/deploymentset-svc/pkg/depset"
)

func ExecuteRequest(m modeler, method, url string, t *testing.T) *httptest.ResponseRecorder {
	server := server{
		model: m,
	}
	server.setupRoutes()

	req, err := http.NewRequest(method, url, nil)
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

	res := ExecuteRequest(m, "GET", fmt.Sprintf("/orgs/%s/apps/%s/sets/%s", orgID, appID, setID), t)

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

	res := ExecuteRequest(m, "GET", fmt.Sprintf("/orgs/%s/apps/%s/sets", orgID, appID), t)

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

	res := ExecuteRequest(m, "GET", fmt.Sprintf("/orgs/%s/apps/%s/sets", orgID, appID), t)

	is.Equal(res.Code, http.StatusOK) // Should return 200

	is.Equal(res.Body.String(), "[]") // Returned Sets should be empty array

}
