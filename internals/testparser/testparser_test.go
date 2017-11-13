package testparser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

var basic = `
=== RUN   TestGetDeviceCount
--- PASS: TestGetDeviceCount (0.07s)
	context_test.go:17: 0 connected devices
=== RUN   TestCreateContext
--- PASS: TestCreateContext (0.00s)
=== RUN   TestGetDevice
--- FAIL: TestGetDevice (0.00s)
	librealsense_test.go:22: no device connected
FAIL
coverage: 9.6% of statements
exit status 1
FAIL	github.com/benhinchley/librealsense-go	0.108s
`

var failedBuild = `
=== RUN   TestGreeting
=== RUN   TestGreeting/strange
--- PASS: TestGreeting (0.00s)
    --- PASS: TestGreeting/strange (0.00s)
PASS
coverage: 100.0% of statements
ok  	github.com/benhinchley/testymctestface	0.006s	coverage: 100.0% of statements
FAIL	github.com/benhinchley/testymctestface/foo [build failed]
`

var failedBuild2 = `
# github.com/benhinchley/pit
./pit_test.go:7:2: imported and not used: "github.com/sanity-io/litter"
FAIL	github.com/benhinchley/pit [build failed]
`

var failedSetup = `
# github.com/benhinchley/pit/internals/testparser
internals/testparser/testparser_test.go:47:15: missing ',' before newline in argument list
internals/testparser/testparser_test.go:48:3: expected operand, found '}'
internals/testparser/testparser_test.go:52:3: missing ',' in argument list
FAIL    github.com/benhinchley/pit/internals/testparser [setup failed]
`

func TestParse(t *testing.T) {
	tests := []string{
		basic,
		failedBuild,
		failedBuild2,
		failedSetup,
	}
	for _, test := range tests {
		var b bytes.Buffer
		fmt.Fprint(&b, test)

		if r, err := Parse(&b); err != nil {
			t.Error(err)
		} else {
			out, _ := json.MarshalIndent(r, "", " ")
			t.Log(string(out))
		}
	}
}
