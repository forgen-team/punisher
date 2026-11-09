package services

import (
	"reflect"
	"testing"
)

func TestStudyPregenerateTargetsPutsRequestedTopicFirst(t *testing.T) {
	cases := []struct {
		name string
		res  Resolution
		want []string
	}{
		{"topic and different prerequisite start", Resolution{ConceptIDs: []string{"os.process", "os.kernel"}, StartingConceptID: "arch.data-representation"}, []string{"os.process", "arch.data-representation"}},
		{"start equals topic", Resolution{ConceptIDs: []string{"os.process"}, StartingConceptID: "os.process"}, []string{"os.process"}},
		{"everything learned", Resolution{ConceptIDs: []string{"os.process"}}, []string{"os.process"}},
		{"nothing resolved", Resolution{}, []string{}},
	}
	for _, tc := range cases {
		if got := studyPregenerateTargets(tc.res); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
