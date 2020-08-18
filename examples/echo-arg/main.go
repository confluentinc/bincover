package main

import (
	"fmt"
	"os"
	"strconv"

        "github.com/lxiaopei/bincover/examples/echo-arg/add"
	"github.com/confluentinc/bincover"
)

var (
	// Injected from linker flags like `go build -ldflags "-X main.version=$VERSION" -X ...`
	isTest = "false"
)

func main() {
	isTest, err := strconv.ParseBool(isTest)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	exitCode := 0
	switch len(os.Args) {
	case 2:
		fmt.Printf("Your argument is \"%s\"\n", os.Args[1])
	case 1:
		fmt.Println("Please provide an argument")
		exitCode = 1
	default:
		panic("More than 2 arguments provided! Ahh!")
	}
	if exitCode != 0 {
		if isTest {
			bincover.ExitCode = exitCode
		} else {
			os.Exit(exitCode)
		}
	}
	add.add(os.Args)
}
