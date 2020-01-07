package test_bin

import (
	"fmt"

	"github.com/confluentinc/bincover"
)

func main() {
	fmt.Println("Hello world")
	bincover.ExitCode = 1
}
