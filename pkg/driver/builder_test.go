package driver

import (
	"fmt"
	"testing"
)

func Test_driver(t *testing.T) {
	b := Builder{}
	invalidVersion := 4
	_, err := b.GetDriver(invalidVersion, "")
	wantErr := fmt.Sprintf("invalid version ( %d )  mentioned in yml file", invalidVersion)
	if err.Error() != wantErr {
		t.Errorf("want %s , got %s", err.Error(), wantErr)
	}
}
