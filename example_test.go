package netx_test

import (
	"fmt"
	"log"
	"time"

	"github.com/bassosimone/netx"
)

func Example() {
	dialer := netx.NewDialer(time.Now())
	conn, err := dialer.Dial("tcp", "www.google.com:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	measurements := dialer.PopMeasurements()
	fmt.Printf("%+v\n", len(measurements) > 0)
	// Output: true
}
