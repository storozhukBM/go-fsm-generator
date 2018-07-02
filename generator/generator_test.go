package generator

import (
	"testing"
	"os"
	"io/ioutil"
	"bytes"
	"fmt"
)

func TestRunGeneratorForTypes(t *testing.T) {
	actualTestOutputFile := "./testdata/some.fsm.go"
	expectedOutputFile := "./testdata/expected/expected.some.fsm.go"

	err := os.Remove(actualTestOutputFile)
	if err != nil && err.Error() != fmt.Sprintf("remove %s: no such file or directory", actualTestOutputFile) {
		t.Errorf("%s file can't be removed: %s", actualTestOutputFile, err.Error())
	}
	RunGeneratorForTypes("./testdata", []string{"SomeDeclaration"}, true)

	expected, err := ioutil.ReadFile(expectedOutputFile)
	if err != nil {
		t.Errorf("can't read expected file: %s", err.Error())
	}

	actual, err := ioutil.ReadFile(actualTestOutputFile)
	if err != nil {
		t.Errorf("can't read actual file: %s", err.Error())
	}
	if !bytes.Equal(actual, expected) {
		t.Errorf("actual `%s` and expected `%s` files deffer", actualTestOutputFile, expectedOutputFile)
	}
}

func TestVerifyTypeNames(t *testing.T) {
	err := verifySpecifiedTypes([]string{"OfLibertyDeclaration"})
	if err != nil {
		t.Errorf("verification should not fail on type `OfLibertyDeclaration`")
	}

	err = verifySpecifiedTypes([]string{"OfLibertyDeclaration", "SomeType"})
	if err == nil {
		t.Errorf("verification should fail on type `SomeType`")
	}
	expected := "unsupported type name. type name should have `Declaration` suffix. type: SomeType"
	if err.Error() != expected {
		t.Errorf("expected {%s}; actual: {%s}", expected, err.Error())
	}
}
