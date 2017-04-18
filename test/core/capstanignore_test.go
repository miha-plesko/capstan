/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package core_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/mikelangelo-project/capstan/core"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type testingCapstanignoreSuite struct{}

var _ = Suite(&testingCapstanignoreSuite{})

func (s *testingCapstanignoreSuite) TestIsIgnored(c *C) {
	m := []struct {
		comment      string
		pattern      string
		path         string
		shouldIgnore bool
	}{
		{
			"fully specified file in root #1",
			"/myfile.txt", "/myfile.txt", true,
		},
		{
			"fully specified file in root #2",
			"/myfile.txt", "/myfolder/myfile.txt", false,
		},

		{
			"fully specified file not in root #1",
			"/myfolder/myfile.txt", "/myfile.txt", false,
		},
		{
			"fully specified file not in root #2",
			"/myfolder/myfile.txt", "/myfolder/myfile.txt", true,
		},
		{
			"file by extension in root #1",
			"/*.txt", "/myfile.txt", true,
		},
		{
			"file by extension in root #2",
			"/*.txt", "/myfolder/myfile.txt", false,
		},
		{
			"file by extension not in root #1",
			"/myfolder/*.txt", "/myfile.txt", false,
		},
		{
			"file by extension not in root #2",
			"/myfolder/*.txt", "/myfolder/myfile.txt", true,
		},
		{
			"file by extension not in root #3",
			"/myfolder/*.txt", "/myfolder/subfolder/myfile.txt", false,
		},
		{
			"fully specified file in any subfolder #1",
			"/**/file.txt", "/myfile.txt", true,
		},
		{
			"fully specified file in any subfolder #2",
			"/**/file.txt", "/myfolder/myfile.txt", true,
		},
		{
			"fully specified file in any subfolder #3",
			"/**/file.txt", "/myfolder/subfolder/myfile.txt", true,
		},
		{
			"whole folder one level #1",
			"/myfolder/*", "/myfolder/myfile.txt", true,
		},
		{
			"whole folder one level #2",
			"/myfolder/*", "/myfolder", false,
		},
		{
			"whole folder one level #3",
			"/myfolder/*", "/myfolder/subfolder/myfile.txt", true,
		},
		{
			"whole folder one level #4",
			"/myfolder/*", "/myfolder/subfolder", true,
		},
		{
			"whole folder two levels #1",
			"/myfolder/subfolder/*", "/myfolder/subfolder", false,
		},
		{
			"whole folder two levels #2",
			"/myfolder/subfolder/*", "/myfolder", false,
		},
		{
			"whole folder two whole levels #1",
			"/myfolder/*/*", "/myfolder", false,
		},
		{
			"whole folder two whole levels #2",
			"/myfolder/*/*", "/myfolder/subfolder", false,
		},
		{
			"whole folder two whole levels #3",
			"/myfolder/*/*", "/myfolder/subfolder/myfile.txt", true,
		},
		{
			"any text file in project #1",
			"/**/*.txt", "/myfile.txt", true,
		},
		{
			"any text file in project #2",
			"/**/*.txt", "/myfolder/myfile.txt", true,
		},
		{
			"any text file in project #3",
			"/**/*.txt", "/myfolder/subfolder/myfile.txt", true,
		},
		{
			"additional test #1",
			"/subfolder/*", "/myfolder/subfolder/myfile.txt", false,
		},
		{
			"additional test #2",
			"/myfolder/*.txt", "/myfolder/myfileXtxt", false,
		},
		{
			"additional test #3",
			"/myfolder", "/myfolder2", false,
		},
		{
			"additional test #4",
			"/myfolder/*", "/myfolder2", false,
		},
		{
			"always ignore /meta",
			"", "/meta", true,
		},
		{
			"always ignore /mpm-pkg",
			"", "/mpm-pkg", true,
		},
		{
			"always ignore /.git",
			"", "/.git", true,
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Setup
		capstanignore, _ := core.CapstanignoreInit("")
		capstanignore.AddPattern(args.pattern)
		capstanignore.CompilePatterns()

		// This is what we're testing here.
		ignoreYesNo := capstanignore.IsIgnored(args.path)

		// Expectations.
		c.Check(ignoreYesNo, Equals, args.shouldIgnore)
	}
}

//
// Run benchmarks like this: `go test ./test/... -bench=. -check.b`
//

// Measured results:
// - having a list of regexp:                      92.176 ns/op
// - having a single regexp (reg1|reg2|reg3|...): 760.719 ns/op
func (s *testingCapstanignoreSuite) BenchmarkIsIgnored(c *C) {
	// Add a lot of non-matching patterns at the beginning
	// and finally a matching pattern at the very end. This
	// way `n` comparisons will be performed for sure.
	n := 1000
	capstanignore, _ := core.CapstanignoreInit("")
	for i := 0; i < n; i++ {
		capstanignore.AddPattern("/this/will/not/match/*")
	}
	capstanignore.AddPattern("/this/is/a/match")
	capstanignore.CompilePatterns()

	path := "/this/is/a/match"
	for i := 0; i < c.N; i++ {
		capstanignore.IsIgnored(path)
	}
}

// Measured results:
// - using "/this/is/a/match" clutter:   92.531 ns/op
// - using "^/this/is/a/match$" clutter: 96.638 ns/op
func (s *testingCapstanignoreSuite) BenchmarkArray(c *C) {
	n := 1000
	var compiledPatterns []*regexp.Regexp
	clutter := "^/this/will/not/match/.*$"
	for i := 0; i < n; i++ {
		compiledPatterns = append(compiledPatterns, regexp.MustCompile(clutter))
	}
	compiledPatterns = append(compiledPatterns, regexp.MustCompile("^/this/is/a/match$"))

	path := "/this/is/a/match"
	for i := 0; i < c.N; i++ {
		isIgnored := false
		for _, pattern := range compiledPatterns {
			isIgnoredCurr := pattern.MatchString(path)
			isIgnored = isIgnored || isIgnoredCurr
		}
		c.Assert(isIgnored, Equals, true)
	}
}

// Measured results:
// - using "/this/is/a/match" clutter:   1.591 ns/op
// - using "^/this/is/a/match$" clutter: 656.719 ns/op
func (s *testingCapstanignoreSuite) BenchmarkOr(c *C) {
	n := 1000
	pattern := ""
	clutter := "^/this/will/not/match/.*$"
	for i := 0; i < n; i++ {
		pattern = pattern + "|" + clutter
	}
	pattern = pattern + "|/this/is/a/match"
	pattern = strings.TrimLeft(pattern, "|")
	compiledPattern := regexp.MustCompile(pattern)

	path := "^/this/is/a/match$"
	for i := 0; i < c.N; i++ {
		isIgnored := compiledPattern.MatchString(path)
		c.Assert(isIgnored, Equals, true)
	}
}
