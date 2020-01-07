// +build testrunmain

package test_bin

import (
	"testing"

	"github.com/confluentinc/bincover"
)

func TestRunMain(t *testing.T) {
	bincover.RunTest(main)
}
