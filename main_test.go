// Copyright 2020 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package main

import (
	"flag"
	"os"
	"testing"
)

var (
	acceptanceTesting bool
)

func init() {
	flag.BoolVar(&acceptanceTesting, "acceptance-testing", false,
		"Acceptance testing mode, starts the application main function "+
			"with cover mode enabled. Non-flag arguments are passed"+
			"to the main application, add '--' after test flags to"+
			"pass flags to main.",
	)
}

func TestMain(m *testing.M) {
	flag.Parse()
	if acceptanceTesting {
		// Override 'run' flags to only execute TestDoMain
		flag.Set("test.run", "TestDoMain")
	}
	os.Exit(m.Run())
}

func TestDoMain(t *testing.T) {
	if !acceptanceTesting {
		t.Skip()
	}
	doMain(append(os.Args[:1], flag.Args()...))
}
