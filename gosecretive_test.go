package gosecretive

import (
	"encoding/json"
	"reflect"
	"testing"
)

type SomeType string

const SomeTypeName SomeType = "SomeTypeName"

type A struct {
	String            string
	Int               int
	MapStrToInterface map[string]interface{}
}
type B struct {
	A
	StringOther string
	TypeString  SomeType
}
type C struct {
	String         string
	MapStrToStr    map[string]string
	Interface      interface{}
	OtherStruct    A
	OtherStructPtr *A
}

type D struct {
	List                      []string
	InitializedListEmpty      []string
	UnInitializedList         []interface{}
	InitializedMapEmpty       map[string]string
	UnInitializedMap          map[string]string
	InitializedStructEmpty    A
	InitializedStructPtrEmpty *A
	UnInitializedStructPtr    *A
	UnInitializedStruct       A
}

var TestCases = []struct {
	name             string
	Raw              interface{}
	Scrubbed         interface{}
	Secretes         map[string]string
	ScrubFuncHandler OnValueFuncHandler
}{
	{
		name: "Struct",
		Raw:  &A{String: "hideMe", Int: 1, MapStrToInterface: map[string]interface{}{"password": "123456"}},
		Scrubbed: &A{
			String:            "scrubbed/String",
			Int:               1,
			MapStrToInterface: map[string]interface{}{"password": "scrubbed/MapStrToInterface/password"},
		},
		Secretes: map[string]string{
			"scrubbed/String":                     "hideMe",
			"scrubbed/MapStrToInterface/password": "123456",
		},
		ScrubFuncHandler: scrubbingFunction([]string{
			"/String",
			"/MapStrToInterface/password",
		}),
	},
	{
		name: "Struct with embedded struct",
		Raw: &B{
			A: A{
				String: "Expose Me", Int: 1, MapStrToInterface: map[string]interface{}{"name": "B/A"},
			},
			StringOther: "hideMe",
			TypeString:  SomeTypeName,
		},
		Scrubbed: &B{
			A: A{
				String: "Expose Me", Int: 1, MapStrToInterface: map[string]interface{}{
					"name": "scrubbed/A/MapStrToInterface/name",
				},
			},
			StringOther: "scrubbed/StringOther",
			TypeString:  SomeTypeName,
		},
		Secretes: map[string]string{
			"scrubbed/A/MapStrToInterface/name": "B/A",
			"scrubbed/StringOther":              "hideMe",
		},
		ScrubFuncHandler: scrubbingFunction([]string{
			"/A/MapStrToInterface/name",
			"/StringOther",
		}),
	},
	{
		name: "Struct containing a reference to another struct",
		Raw: &C{
			String:      "C",
			MapStrToStr: map[string]string{"content": "private!!", "name": "pemfile"},
			Interface:   map[string]interface{}{"content": "dontell", "name": "secret"},
			OtherStruct: A{String: "A", Int: 1, MapStrToInterface: map[string]interface{}{
				"name": "C/OtherStruct",
			}},
			OtherStructPtr: &A{String: "&A", Int: 1, MapStrToInterface: map[string]interface{}{
				"name": "C/OtherStructPtr",
			}},
		},

		Scrubbed: &C{
			String:      "scrubbed/String",
			MapStrToStr: map[string]string{"content": "scrubbed/MapStrToStr/content", "name": "pemfile"},
			Interface:   map[string]interface{}{"content": "scrubbed/Interface/content", "name": "secret"},
			OtherStruct: A{String: "A", Int: 1, MapStrToInterface: map[string]interface{}{
				"name": "scrubbed/OtherStruct/MapStrToInterface/name",
			}},
			OtherStructPtr: &A{String: "&A", Int: 1, MapStrToInterface: map[string]interface{}{
				"name": "scrubbed/OtherStructPtr/MapStrToInterface/name",
			}},
		},
		Secretes: map[string]string{
			"scrubbed/String":                                "C",
			"scrubbed/MapStrToStr/content":                   "private!!",
			"scrubbed/Interface/content":                     "dontell",
			"scrubbed/OtherStruct/MapStrToInterface/name":    "C/OtherStruct",
			"scrubbed/OtherStructPtr/MapStrToInterface/name": "C/OtherStructPtr",
		},
		ScrubFuncHandler: scrubbingFunction([]string{
			"/String",
			"/MapStrToStr/content",
			"/Interface/content",
			"/OtherStruct/MapStrToInterface/name",
			"/OtherStructPtr/MapStrToInterface/name",
		}),
	},
	{
		name: "Struct containing a list",
		Raw: &D{
			List:                      []string{"a", "b", "c"},
			InitializedListEmpty:      []string{},
			InitializedMapEmpty:       map[string]string{},
			InitializedStructEmpty:    A{},
			InitializedStructPtrEmpty: &A{},
		},
		Scrubbed: &D{
			List:                      []string{"scrubbed/List[0]", "b", "scrubbed/List[2]"},
			InitializedListEmpty:      []string{},
			InitializedMapEmpty:       map[string]string{},
			InitializedStructEmpty:    A{},
			InitializedStructPtrEmpty: &A{},
		},
		Secretes: map[string]string{
			"scrubbed/List[0]": "a",
			"scrubbed/List[2]": "c",
		},
		ScrubFuncHandler: scrubbingFunction([]string{
			"/List[0]",
			"/List[2]",
		}),
	},
}

func TestRestore(t *testing.T) {
	type args struct {
		objectToRestore interface{}
		secrets         map[string]string
	}
	type testCase struct {
		name                   string
		args                   args
		expectedRestoredObject interface{}
	}
	var testCases []testCase
	for _, tc := range TestCases {
		testCases = append(testCases, testCase{
			name: tc.name,
			args: args{
				objectToRestore: tc.Scrubbed,
				secrets:         tc.Secretes,
			},
			expectedRestoredObject: tc.Raw,
		})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// Restore should be idempotent, run it few times
			var restoredObject interface{}
			for i := 0; i < 3; i++ {
				t.Log("Iteration", i)

				// marshal testCase.args.objectToRestore to JSON
				preRestoreMarshaledObject, err := json.Marshal(tc.args.objectToRestore)
				if err != nil {
					t.Fatal(err)
				}

				restoredObject = Restore(tc.args.objectToRestore, tc.args.secrets)
				if !reflect.DeepEqual(restoredObject, tc.expectedRestoredObject) {
					t.Errorf("Restore() restoredObject = %v, expectedRestoredObject %v", restoredObject, tc.expectedRestoredObject)
				}

				postRestoreMarshaledObject, err := json.Marshal(tc.args.objectToRestore)
				if err != nil {
					t.Fatal(err)
				}

				// verify that the original object was not modified
				if string(preRestoreMarshaledObject) != string(postRestoreMarshaledObject) {
					t.Errorf("Restore() should not modify the original object, preScrubMarshaledObjectToScrub = %v, postScrubMarshaledObjectToScrub %v", preRestoreMarshaledObject, postRestoreMarshaledObject)
				}

				// after first iteration, we want to make sure that the object to restore did not change
				tc.args.objectToRestore = restoredObject

			}
		})
	}
}

func TestScrub(t *testing.T) {
	type args struct {
		objectToScrub    interface{}
		scrubFuncHandler OnValueFuncHandler
	}
	type testCase struct {
		name                   string
		args                   args
		expectedScrubbedObject interface{}
		secrets                map[string]string
	}
	var testCases []testCase
	for _, tc := range TestCases {
		testCases = append(testCases, testCase{
			name: tc.name,
			args: args{
				objectToScrub:    tc.Raw,
				scrubFuncHandler: tc.ScrubFuncHandler,
			},
			expectedScrubbedObject: tc.Scrubbed,
			secrets:                tc.Secretes,
		})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// Scrub should be idempotent, run it few times
			var scrubbedObject interface{}
			var secrets map[string]string
			for i := 0; i < 3; i++ {
				t.Log("Scrubbing iteration", i)

				// marshal tc.args.objectToRestore to JSON
				preScrubMarshaledObjectToScrub, err := json.Marshal(tc.args.objectToScrub)
				if err != nil {
					t.Fatal(err)
				}

				scrubbedObject, secrets = Scrub(tc.args.objectToScrub, tc.args.scrubFuncHandler)
				if !reflect.DeepEqual(scrubbedObject, tc.expectedScrubbedObject) {
					t.Errorf("Scrub() scrubbedObject = %v, expectedRestoredObject %v", scrubbedObject, tc.expectedScrubbedObject)
				}

				postScrubMarshaledObjectToScrub, err := json.Marshal(tc.args.objectToScrub)
				if err != nil {
					t.Fatal(err)
				}

				// verify that the original object was not modified
				if string(preScrubMarshaledObjectToScrub) != string(postScrubMarshaledObjectToScrub) {
					t.Errorf("Scrub() should not modify the original object, preScrubMarshaledObjectToScrub = %v, postScrubMarshaledObjectToScrub %v", preScrubMarshaledObjectToScrub, postScrubMarshaledObjectToScrub)
				}

				if !reflect.DeepEqual(secrets, tc.secrets) {
					t.Errorf("Scrub() secrets = %v, expectedRestoredObject %v", secrets, tc.secrets)
				}

				// after first iteration, we want to make sure that the object to scrub did not change
				tc.args.objectToScrub = scrubbedObject

				// after first iteration, we want to make sure that the secrets generated is empty (aka nothing was scrubbed)
				tc.secrets = map[string]string{}
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	for _, tc := range TestCases {
		t.Run(tc.name, func(t *testing.T) {
			raw := tc.Raw
			for i := 0; i < 3; i++ {
				restoredObject := Restore(Scrub(raw, tc.ScrubFuncHandler))
				if !reflect.DeepEqual(restoredObject, tc.Raw) {
					t.Errorf("RoundTrip() restoredObject = %v, expectedRestoredObject %v", restoredObject, tc.Raw)
				}
				raw = restoredObject
			}
		})
	}
}

func scrubbingFunction(allowedFieldPaths []string) OnValueFuncHandler {
	return func(fieldPath string, valueToScrub interface{}) *string {
		if stringInSlice(fieldPath, allowedFieldPaths) {
			s := "scrubbed" + fieldPath
			return &s
		}
		return nil
	}
}

func stringInSlice(s string, ss []string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
