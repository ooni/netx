// Package testingx contains testing extensions
package testingx

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bassosimone/netx/model"
)

// SpawnLogger spawns a goroutine that logs measurements on the stdout.
func SpawnLogger(in chan model.Measurement) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case m := <-in:
				data, err := json.Marshal(m)
				if err != nil {
					panic(err)
				}
				fmt.Printf("%s\n", string(data))
			}
		}
	}()
	return cancel
}
