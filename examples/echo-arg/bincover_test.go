// +build testbincover

package main

import (
	"testing"

	"github.com/confluentinc/bincover"
)

func TestBincover(t *testing.T) {
	bincover.RunTest(main)
}
