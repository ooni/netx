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
	var connid int64
	if err = netx.GetConnID(conn, &connid); err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	measurements := dialer.PopMeasurements()
	fmt.Printf("%d %+v\n", connid, len(measurements) > 0)
	// Output: 1 true
}
