// +build ignore

package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
)

var (
	printFlag = flag.Bool("print", false, "Print true or false, and always exit with 0 except in case of usage errors")
)

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s <minimum-version>\n", os.Args)
		os.Exit(2)
	}

	re := regexp.MustCompile(`^(?:go)?(\d+).(\d+)(?:\.(\d+))?$`)
	arg := flag.Arg(0)
	argSubmatch := re.FindStringSubmatch(arg)
	if argSubmatch == nil {
		fmt.Fprintln(os.Stderr, "Invalid minimum-version: expected x.y or x.y.z")
		os.Exit(2)
	}

	runtimeVersion := runtime.Version()
	goSubmatch := re.FindStringSubmatch(runtimeVersion)
	if goSubmatch == nil {
		fmt.Fprintln(os.Stderr, "Failed to parse runtime.Version(%s)", runtimeVersion)
		os.Exit(3)
	}

	result := true
	minVersion := makeInts(argSubmatch[1:])
	goVersion := makeInts(goSubmatch[1:])
	for i := range minVersion {
		n := goVersion[i] - minVersion[i]
		if n < 0 {
			if *printFlag {
				result = false
			} else {
				fmt.Fprintf(os.Stderr, "%s < %s\n", runtimeVersion, arg)
				os.Exit(1)
			}
		}
		if n > 0 {
			break
		}
	}
	if *printFlag {
		fmt.Println(result)
	}
}

func makeInts(s []string) []int {
	ints := make([]int, len(s))
	for i, s := range s {
		if s == "" {
			s = "0"
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			panic(err)
		}
		ints[i] = n
	}
	return ints
}
