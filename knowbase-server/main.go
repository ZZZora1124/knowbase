// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

// Package main is special.  It defines a
// standalone executable program, not a library.
// Within package main the function main is also
// special—it's where execution of the program begins.
// Whatever main does is what the program does.
package main

import (
	"fmt"
	"os"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/cmd"
)

// main the function where execution of the program begins
func main() {
	if err := cmd.NewRootCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "root command execute err: %v\n", err)
		os.Exit(1)
	}
}


