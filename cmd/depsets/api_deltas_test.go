package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/matryer/is"
	"humanitec.io/deploymentset-svc/pkg/depset"
)

func orderInvarientEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aSorted := make([]string, len(a))
	bSorted := make([]string, len(b))

	copy(aSorted, a)
	copy(bSorted, b)

	sort.Strings(aSorted)
	sort.Strings(bSorted)

	return reflect.DeepEqual(aSorted, bSorted)
}

type matchingDeltaMetadata struct{ m DeltaMetadata }

func IgnoreDateMetadata(m DeltaMetadata) gomock.Matcher {
	return &matchingDeltaMetadata{m}
}

func (m *matchingDeltaMetadata) String() string {
	return fmt.Sprintf("%v", m.m)
}

func (m *matchingDeltaMetadata) Matches(x interface{}) bool {
	metadataToTest, ok := x.(DeltaMetadata)
	if !ok {
		return false
	}

	return orderInvarientEqual(m.m.Contributers, metadataToTest.Contributers) &&
		m.m.CreatedBy == metadataToTest.CreatedBy
}

func TestDeltaForEmptyInputs(t *testing.T) {
	is := is.New(t)
	res := ExecuteRequest(nil, "POST", "/orgs/test-org/apps/test-app/deltas", nil, t)
	is.Equal(res.Code, http.StatusUnprocessableEntity) // Should return 422: POST /orgs/test-org/apps/test-app/deltas

	res = ExecuteRequest(nil, "PUT", "/orgs/test-org/apps/test-app/deltas/DELTAID", nil, t)
	is.Equal(res.Code, http.StatusUnprocessableEntity) // Should return 422: PUT /orgs/test-org/apps/test-app/deltas/DELTAID

	res = ExecuteRequest(nil, "PATCH", "/orgs/test-org/apps/test-app/deltas/DELTAID", nil, t)
	is.Equal(res.Code, http.StatusUnprocessableEntity) // Should return 422: PATCH /orgs/test-org/apps/test-app/deltas/DELTAID
}

func TestDeltaForMalformedInputs(t *testing.T) {
	is := is.New(t)
	invalidJSON := bytes.NewBuffer([]byte(`THIS IS NOT VALID JSON!`))
	res := ExecuteRequest(nil, "POST", "/orgs/test-org/apps/test-app/deltas", invalidJSON, t)
	is.Equal(res.Code, http.StatusUnprocessableEntity) // Should return 422: POST /orgs/test-org/apps/test-app/deltas

	res = ExecuteRequest(nil, "PUT", "/orgs/test-org/apps/test-app/deltas/DELTAID", invalidJSON, t)
	is.Equal(res.Code, http.StatusUnprocessableEntity) // Should return 422: PUT /orgs/test-org/apps/test-app/deltas/DELTAID

	res = ExecuteRequest(nil, "PATCH", "/orgs/test-org/apps/test-app/deltas/DELTAID", invalidJSON, t)
	is.Equal(res.Code, http.StatusUnprocessableEntity) // Should return 422: PATCH /orgs/test-org/apps/test-app/deltas/DELTAID}
}

func TestGetDelta(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	deltaID := "0123456789ABCDEFDEADBEEFDEADBEEFDEADBEEF"
	createdBy := "test-user"
	expectedDeltaWrapper := DeltaWrapper{
		Metadata: DeltaMetadata{
			CreatedAt:      time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
			CreatedBy:      createdBy,
			LastModifiedAt: time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
		},
		Content: depset.Delta{
			Modules: depset.ModuleDeltas{
				Add: map[string]map[string]interface{}{
					"test-module": map[string]interface{}{
						"version": "TEST_VERSION",
					},
				},
			},
		},
	}

	m.
		EXPECT().
		selectDelta(orgID, appID, deltaID).
		Return(expectedDeltaWrapper, nil).
		Times(1)

	res := ExecuteRequest(m, "GET", fmt.Sprintf("/orgs/%s/apps/%s/deltas/%s", orgID, appID, deltaID), nil, t)

	is.Equal(res.Code, http.StatusOK) // Should return 200

	var returnedDeltaWrapper DeltaWrapper
	json.Unmarshal(res.Body.Bytes(), &returnedDeltaWrapper)

	is.Equal(returnedDeltaWrapper, expectedDeltaWrapper) // Returned Delta should match initial delta

}

func TestGetAllDeltas(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	expectedDeltaWrappers := []DeltaWrapper{
		DeltaWrapper{
			ID: "0123456789ABCDEFDEADBEEFDEADBEEFDEADBEEF",
			Metadata: DeltaMetadata{
				CreatedAt:      time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
				CreatedBy:      "user-01",
				LastModifiedAt: time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
			},
			Content: depset.Delta{
				Modules: depset.ModuleDeltas{
					Add: map[string]map[string]interface{}{
						"test-module": map[string]interface{}{
							"version": "TEST_VERSION",
						},
					},
				},
			},
		},
		DeltaWrapper{
			ID: "DEADBEEFDEADBEEFDEADBEEF0123456789ABCDEF",
			Metadata: DeltaMetadata{
				CreatedAt:      time.Date(2020, time.January, 1, 2, 0, 0, 0, time.UTC),
				CreatedBy:      "user-02",
				LastModifiedAt: time.Date(2020, time.January, 1, 2, 0, 0, 0, time.UTC),
			},
			Content: depset.Delta{
				Modules: depset.ModuleDeltas{
					Add: map[string]map[string]interface{}{
						"test-module": map[string]interface{}{
							"version": "TEST_VERSION02",
						},
					},
				},
			},
		},
	}

	m.
		EXPECT().
		selectAllDeltas(orgID, appID).
		Return(expectedDeltaWrappers, nil).
		Times(1)

	res := ExecuteRequest(m, "GET", fmt.Sprintf("/orgs/%s/apps/%s/deltas", orgID, appID), nil, t)

	is.Equal(res.Code, http.StatusOK) // Should return 200

	var returnedDeltaWrappers []DeltaWrapper
	json.Unmarshal(res.Body.Bytes(), &returnedDeltaWrappers)

	is.Equal(returnedDeltaWrappers, expectedDeltaWrappers) // Returned Delta should match initial delta

}

func TestGetAllDeltas_NoneExist(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"

	m.
		EXPECT().
		selectAllDeltas(orgID, appID).
		Return(nil, nil).
		Times(1)

	res := ExecuteRequest(m, "GET", fmt.Sprintf("/orgs/%s/apps/%s/deltas", orgID, appID), nil, t)

	is.Equal(res.Code, http.StatusOK) // Should return 200

	is.Equal(res.Body.String(), "[]") // Empty array should be returned.

}

func TestCreateDelta(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	deltaID := "0123456789ABCDEFDEADBEEFDEADBEEFDEADBEEF"
	// createdBy := "test-user"
	createdBy := "UNKNOWN"
	userProvidedDelta := depset.Delta{
		Modules: depset.ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"test-module": map[string]interface{}{
					"version": "TEST_VERSION",
				},
			},
		},
	}
	expecetdMetadata := DeltaMetadata{
		CreatedAt:      time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
		CreatedBy:      createdBy,
		LastModifiedAt: time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
	}

	m.
		EXPECT().
		insertDelta(orgID, appID, false, IgnoreDateMetadata(expecetdMetadata), userProvidedDelta).
		Return(deltaID, nil).
		Times(1)

	buf, err := json.Marshal(userProvidedDelta)
	is.NoErr(err)
	body := bytes.NewBuffer(buf)

	res := ExecuteRequest(m, "POST", fmt.Sprintf("/orgs/%s/apps/%s/deltas", orgID, appID), body, t)

	is.Equal(res.Code, http.StatusOK) // Should return 200

	var returnedDeltaID string
	json.Unmarshal(res.Body.Bytes(), &returnedDeltaID)

	is.Equal(returnedDeltaID, deltaID) // Returned ID should match generated ID

}

func TestReplaceDelta(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	deltaID := "0123456789ABCDEFDEADBEEFDEADBEEFDEADBEEF"
	// updatedBy := "test-user"
	updatedBy := "UNKNOWN"
	previousDelta := depset.Delta{
		Modules: depset.ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"test-module": map[string]interface{}{
					"version": "TEST_VERSION",
				},
			},
		},
	}
	previousMetadata := DeltaMetadata{
		CreatedAt:      time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
		CreatedBy:      "previous-user",
		LastModifiedAt: time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
		Contributers:   []string{},
	}
	userProvidedDelta := depset.Delta{
		Modules: depset.ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"test-module": map[string]interface{}{
					"version": "TEST_VERSION02",
				},
			},
		},
	}
	expecetdMetadata := DeltaMetadata{
		CreatedAt:      time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
		CreatedBy:      "previous-user",
		LastModifiedAt: time.Date(2020, time.January, 1, 2, 0, 0, 0, time.UTC),
		Contributers:   []string{updatedBy},
	}

	m.
		EXPECT().
		selectDelta(orgID, appID, deltaID).
		Return(DeltaWrapper{
			ID:       deltaID,
			Content:  previousDelta,
			Metadata: previousMetadata,
		}, nil).
		Times(1)

	m.
		EXPECT().
		updateDelta(orgID, appID, deltaID, false, IgnoreDateMetadata(expecetdMetadata), userProvidedDelta).
		Return(nil).
		Times(1)

	buf, err := json.Marshal(userProvidedDelta)
	is.NoErr(err)
	body := bytes.NewBuffer(buf)

	res := ExecuteRequest(m, "PUT", fmt.Sprintf("/orgs/%s/apps/%s/deltas/%s", orgID, appID, deltaID), body, t)

	is.Equal(res.Code, http.StatusNoContent) // Should return 204

}

func TestReplaceDelta_SameUser(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	deltaID := "0123456789ABCDEFDEADBEEFDEADBEEFDEADBEEF"
	// updatedBy := "test-user"
	currentUser := "UNKNOWN"
	previousDelta := depset.Delta{
		Modules: depset.ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"test-module": map[string]interface{}{
					"version": "TEST_VERSION",
				},
			},
		},
	}
	previousMetadata := DeltaMetadata{
		CreatedAt:      time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
		CreatedBy:      "previous-user",
		LastModifiedAt: time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
		Contributers:   []string{"different-user", currentUser},
	}
	userProvidedDelta := depset.Delta{
		Modules: depset.ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"test-module": map[string]interface{}{
					"version": "TEST_VERSION02",
				},
			},
		},
	}
	expecetdMetadata := DeltaMetadata{
		CreatedAt:      time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
		CreatedBy:      "previous-user",
		LastModifiedAt: time.Date(2020, time.January, 1, 2, 0, 0, 0, time.UTC),
		Contributers:   []string{"different-user", currentUser},
	}

	m.
		EXPECT().
		selectDelta(orgID, appID, deltaID).
		Return(DeltaWrapper{
			ID:       deltaID,
			Content:  previousDelta,
			Metadata: previousMetadata,
		}, nil).
		Times(1)

	m.
		EXPECT().
		updateDelta(orgID, appID, deltaID, false, IgnoreDateMetadata(expecetdMetadata), userProvidedDelta).
		Return(nil).
		Times(1)

	buf, err := json.Marshal(userProvidedDelta)
	is.NoErr(err)
	body := bytes.NewBuffer(buf)

	res := ExecuteRequest(m, "PUT", fmt.Sprintf("/orgs/%s/apps/%s/deltas/%s", orgID, appID, deltaID), body, t)

	is.Equal(res.Code, http.StatusNoContent) // Should return 204

}

func TestReplaceDelta_DeltaDoesNotExist(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	deltaID := "0123456789ABCDEFDEADBEEFDEADBEEFDEADBEEF"

	userProvidedDelta := depset.Delta{
		Modules: depset.ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"test-module": map[string]interface{}{
					"version": "TEST_VERSION",
				},
			},
		},
	}

	m.
		EXPECT().
		selectDelta(orgID, appID, deltaID).
		Return(DeltaWrapper{}, ErrNotFound).
		Times(1)

	buf, err := json.Marshal(userProvidedDelta)
	is.NoErr(err)
	body := bytes.NewBuffer(buf)

	res := ExecuteRequest(m, "PUT", fmt.Sprintf("/orgs/%s/apps/%s/deltas/%s", orgID, appID, deltaID), body, t)

	is.Equal(res.Code, http.StatusNotFound) // Should return 404

}

func TestUpdateDelta_DeltaDoesNotExist(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	deltaID := "0123456789ABCDEFDEADBEEFDEADBEEFDEADBEEF"

	userProvidedDelta := depset.Delta{
		Modules: depset.ModuleDeltas{
			Add: map[string]map[string]interface{}{
				"test-module": map[string]interface{}{
					"version": "TEST_VERSION",
				},
			},
		},
	}

	m.
		EXPECT().
		selectDelta(orgID, appID, deltaID).
		Return(DeltaWrapper{}, ErrNotFound).
		Times(1)

	buf, err := json.Marshal([]depset.Delta{userProvidedDelta})
	is.NoErr(err)
	body := bytes.NewBuffer(buf)

	res := ExecuteRequest(m, "PATCH", fmt.Sprintf("/orgs/%s/apps/%s/deltas/%s", orgID, appID, deltaID), body, t)

	is.Equal(res.Code, http.StatusNotFound) // Should return 404

}

func TestUpdateDelta(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	deltaID := "0123456789ABCDEFDEADBEEFDEADBEEFDEADBEEF"
	currentUser := "UNKNOWN"

	baseDeltaWrapper := DeltaWrapper{
		ID: deltaID,
		Metadata: DeltaMetadata{
			CreatedBy:      "first-user",
			CreatedAt:      time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
			LastModifiedAt: time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
			Contributers:   []string{},
		},
		Content: depset.Delta{
			Modules: depset.ModuleDeltas{
				Add: map[string]map[string]interface{}{
					"test-module": map[string]interface{}{
						"version": "TEST_VERSION",
					},
				},
			},
		},
	}
	deltas := []depset.Delta{
		depset.Delta{
			Modules: depset.ModuleDeltas{
				Add: map[string]map[string]interface{}{
					"second-module": map[string]interface{}{
						"version": "SECOND_VERSION",
					},
				},
			},
		},
	}
	expectedDeltaWrapper := DeltaWrapper{
		ID: deltaID,
		Metadata: DeltaMetadata{
			CreatedBy:      "first-user",
			CreatedAt:      time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
			LastModifiedAt: time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
			Contributers:   []string{currentUser},
		},
		Content: depset.Delta{
			Modules: depset.ModuleDeltas{
				Add: map[string]map[string]interface{}{
					"test-module": map[string]interface{}{
						"version": "TEST_VERSION",
					},
					"second-module": map[string]interface{}{
						"version": "SECOND_VERSION",
					},
				},
				Remove: []string{},
				Update: map[string][]depset.UpdateAction{},
			},
		},
	}

	m.
		EXPECT().
		selectDelta(orgID, appID, deltaID).
		Return(baseDeltaWrapper, nil).
		Times(1)

	m.
		EXPECT().
		updateDelta(orgID, appID, deltaID, false, IgnoreDateMetadata(expectedDeltaWrapper.Metadata), expectedDeltaWrapper.Content).
		Return(nil).
		Times(1)

	buf, err := json.Marshal(deltas)
	is.NoErr(err)
	body := bytes.NewBuffer(buf)

	res := ExecuteRequest(m, "PATCH", fmt.Sprintf("/orgs/%s/apps/%s/deltas/%s", orgID, appID, deltaID), body, t)

	is.Equal(res.Code, http.StatusOK) // Should return 200

	var returnedDeltaWrapper DeltaWrapper
	json.Unmarshal(res.Body.Bytes(), &returnedDeltaWrapper)

	is.True(IgnoreDateMetadata(returnedDeltaWrapper.Metadata).Matches(expectedDeltaWrapper.Metadata)) // Returned Metadata should match expected metadata
	is.Equal(returnedDeltaWrapper.Content, expectedDeltaWrapper.Content)                              // Returned Delta should match expected delta

}

func TestUpdateDelta_EmptyPayload(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockmodeler(ctrl)

	orgID := "test-org"
	appID := "test-app"
	deltaID := "0123456789ABCDEFDEADBEEFDEADBEEFDEADBEEF"

	expectedDeltaWrapper := DeltaWrapper{
		ID: deltaID,
		Metadata: DeltaMetadata{
			CreatedBy:      "first-user",
			CreatedAt:      time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
			LastModifiedAt: time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
			Contributers:   []string{},
		},
		Content: depset.Delta{
			Modules: depset.ModuleDeltas{
				Add: map[string]map[string]interface{}{
					"test-module": map[string]interface{}{
						"version": "TEST_VERSION",
					},
				},
			},
		},
	}
	deltas := []depset.Delta{}

	m.
		EXPECT().
		selectDelta(orgID, appID, deltaID).
		Return(expectedDeltaWrapper, nil).
		Times(1)

	buf, err := json.Marshal(deltas)
	is.NoErr(err)
	body := bytes.NewBuffer(buf)

	res := ExecuteRequest(m, "PATCH", fmt.Sprintf("/orgs/%s/apps/%s/deltas/%s", orgID, appID, deltaID), body, t)

	is.Equal(res.Code, http.StatusOK) // Should return 200

	var returnedDeltaWrapper DeltaWrapper
	json.Unmarshal(res.Body.Bytes(), &returnedDeltaWrapper)

	is.True(IgnoreDateMetadata(returnedDeltaWrapper.Metadata).Matches(expectedDeltaWrapper.Metadata)) // Returned Metadata should match expected metadata
	is.Equal(returnedDeltaWrapper.Content, expectedDeltaWrapper.Content)                              // Returned Delta should match expected delta

}
