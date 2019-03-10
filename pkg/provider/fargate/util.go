package fargate

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ecs"
)

func sliceWalk(windowSz int, input []*string, fn func([]*string) error) error {
	if windowSz < 1 {
		return fmt.Errorf("bad window size")
	}

	cnt := len(input)
	for i := 0; i < cnt; i += windowSz {
		err := fn(input[i : i+min(windowSz, cnt-i)])
		if err != nil {
			return err
		}
	}
	return nil
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}

func keys(in map[string][]*ecs.Task) []string {
	keys := []string{}
	for k := range in {
		keys = append(keys, k)
	}
	return keys
}
