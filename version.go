package main

import (
	fmt "github.com/jhunt/go-ansi"
	"strconv"
	"strings"

	"github.com/cloudfoundry/cli/plugin"
)

var Version string

func vnum(s string) ([]int, error) {
	n := strings.Split(Version, ".")
	ints := make([]int, len(n))
	for i, x := range n {
		num, err := strconv.Atoi(x)
		if err != nil {
			return ints, err
		}
		if num < 0 {
			return ints, fmt.Errorf("Invalid version component '%s'", x)
		}
		ints[i] = num
	}
	return ints, nil
}

func getVersion(s string) (v plugin.VersionType) {
	n, err := vnum(s)
	if err != nil {
		return
	}
	if len(n) >= 1 {
		v.Major = n[0]
	}
	if len(n) >= 2 {
		v.Minor = n[1]
	}
	if len(n) >= 3 {
		v.Build = n[2]
	}
	return
}
