// +build testrunmain

package input_bin

import (
	"testing"

	"github.com/confluentinc/bincover"
)

func TestRunMain(t *testing.T) {
	bincover.RunTest(main)
}
