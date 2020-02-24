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
				Add: map[string]depset.ModuleSpec{
					"test-module": depset.ModuleSpec{
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
					Add: map[string]depset.ModuleSpec{
						"test-module": depset.ModuleSpec{
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
					Add: map[string]depset.ModuleSpec{
						"test-module": depset.ModuleSpec{
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
			Add: map[string]depset.ModuleSpec{
				"test-module": depset.ModuleSpec{
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
