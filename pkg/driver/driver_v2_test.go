package driver

import (
	"reflect"
	"testing"
)

type testArgs struct {
	name          string
	subModulePath string
	diffMap       map[string]int
	wantDiffMap   map[string]int
}

func TestGetSubmoduleBasedDiff(t *testing.T) {
	tests :=
		[]testArgs{
			{
				name:          "test with subModule package included in diff 1",
				subModulePath: "./package/subModule-1",
				diffMap: map[string]int{
					"package/subModule-1/test/testFile1.js": 1,
					"package/subModule-1/test/testFile2.js": 2,
					"package/subModule-2/test/testFile1.js": 3,
					"package/subModule-2/test/testFile2.js": 4,
				},
				wantDiffMap: map[string]int{
					"test/testFile1.js":                     1,
					"test/testFile2.js":                     2,
					"package/subModule-2/test/testFile1.js": 3,
					"package/subModule-2/test/testFile2.js": 4,
				},
			},

			{
				name:          "test with subModule package included in diff 2",
				subModulePath: "package/subModule-1",
				diffMap: map[string]int{
					"package/subModule-1/test/testFile1.js": 1,
					"package/subModule-1/test/testFile2.js": 2,
					"package/subModule-2/test/testFile1.js": 3,
					"package/subModule-2/test/testFile2.js": 4,
				},
				wantDiffMap: map[string]int{
					"test/testFile1.js":                     1,
					"test/testFile2.js":                     2,
					"package/subModule-2/test/testFile1.js": 3,
					"package/subModule-2/test/testFile2.js": 4,
				},
			},
			{
				name:          "test with subModule package included in diff 3",
				subModulePath: "./package/subModule-1/",
				diffMap: map[string]int{
					"package/subModule-1/test/testFile1.js": 1,
					"package/subModule-1/test/testFile2.js": 2,
					"package/subModule-2/test/testFile1.js": 3,
					"package/subModule-2/test/testFile2.js": 4,
				},
				wantDiffMap: map[string]int{
					"test/testFile1.js":                     1,
					"test/testFile2.js":                     2,
					"package/subModule-2/test/testFile1.js": 3,
					"package/subModule-2/test/testFile2.js": 4,
				},
			},
			{
				name:          "test with subModule package included in diff 4",
				subModulePath: "package/subModule-1/",
				diffMap: map[string]int{
					"package/subModule-1/test/testFile1.js": 1,
					"package/subModule-1/test/testFile2.js": 2,
					"package/subModule-2/test/testFile1.js": 3,
					"package/subModule-2/test/testFile2.js": 4,
				},
				wantDiffMap: map[string]int{
					"test/testFile1.js":                     1,
					"test/testFile2.js":                     2,
					"package/subModule-2/test/testFile1.js": 3,
					"package/subModule-2/test/testFile2.js": 4,
				},
			},
			{
				name:          "test with subModule package not included in diff ",
				subModulePath: "package/subModule-1/",
				diffMap: map[string]int{
					"package/subModule-2/test/testFile1.js": 1,
					"package/subModule-2/test/testFile2.js": 2,
					"package/subModule-2/test/testFile3.js": 3,
					"package/subModule-2/test/testFile4.js": 4,
				},
				wantDiffMap: map[string]int{
					"package/subModule-2/test/testFile1.js": 1,
					"package/subModule-2/test/testFile2.js": 2,
					"package/subModule-2/test/testFile3.js": 3,
					"package/subModule-2/test/testFile4.js": 4,
				},
			},
		}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualMap := GetSubmoduleBasedDiff(test.diffMap, test.subModulePath)
			if !reflect.DeepEqual(actualMap, test.wantDiffMap) {
				t.Errorf("not equal wanted %+v , got %+v", test.diffMap, actualMap)
			}
		})
	}
}
