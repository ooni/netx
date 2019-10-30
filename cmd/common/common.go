// Package common contains common flags
package common

import "flag"

var (
	// FlagHelp is used to request the help screen
	FlagHelp = flag.Bool("help", false, "Print usage")

	// FlagSNI forces using a specific SNI
	FlagSNI = flag.String("sni", "", "Force specific SNI usage")
)
