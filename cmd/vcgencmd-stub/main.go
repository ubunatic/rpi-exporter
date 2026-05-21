// vcgencmd-stub mimics the vcgencmd tool for local testing on non-RPi systems.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: vcgencmd <command> [args...]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "measure_volts":
		fmt.Println("volt=1.2000V")
	case "get_throttled":
		fmt.Println("throttled=0x0")
	case "measure_temp":
		fmt.Println("temp=45.0'C")
	case "measure_clock":
		fmt.Println("frequency(0)=700000000")
	case "get_mem":
		id := "arm"
		if len(os.Args) >= 3 {
			id = os.Args[2]
		}
		fmt.Printf("%s=512M\n", id)
	case "get_rsts":
		fmt.Println("get_rsts=1")
	default:
		fmt.Fprintf(os.Stderr, "vcgencmd-stub: unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
