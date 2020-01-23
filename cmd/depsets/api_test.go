package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"humanitec.io/deploymentset-svc/pkg/depset"
)

func TestGetSet(t *testing.T) {
	ctrl := gomock.NewController(t)

	// Assert that Bar() is invoked.
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

	server := server{
		model: m,
	}
	server.setupRoutes()

	req, err := http.NewRequest("GET", fmt.Sprintf("/orgs/%s/apps/%s/sets/%s", orgID, appID, setID), nil)
	if err != nil {
		t.Errorf("creating request : %v", err)
	}
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got : %v", w.Code)
	}

	var returnedSetWrapper SetWrapper
	json.Unmarshal(w.Body.Bytes(), &returnedSetWrapper)

	if !reflect.DeepEqual(returnedSetWrapper, expectedSetWrapper) {
		t.Errorf("Expected: `%+v`, got `%+v`", expectedSetWrapper, returnedSetWrapper)
	}

}
