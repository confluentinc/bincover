package add

import (
)

func add(args []string) string {
	result := "hello "
        for _, arg := range args {
	    result += arg
}
	return result
}
