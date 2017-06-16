// Copyright 2015 The rkt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/appc/spec/schema/types"
	rktlog "github.com/rkt/rkt/pkg/log"
	stage1initcommon "github.com/rkt/rkt/stage1/init/common"
)

var (
	debug   bool
	podPid  string
	appName string

	log  *rktlog.Logger
	diag *rktlog.Logger
)

func init() {
	flag.BoolVar(&debug, "debug", false, "Run in debug mode")
	flag.StringVar(&podPid, "pid", "", "Pod PID")
	flag.StringVar(&appName, "appname", "", "Application name")

	log, diag, _ = rktlog.NewLogSet("fly-enter", false)
}

func getRootDir(pid string) (string, error) {
	rootLink := fmt.Sprintf("/proc/%s/root", pid)

	return os.Readlink(rootLink)
}

// readEnv reads the environment from the env file written by stage1 run
func readEnv() ([]string, error) {
	var env []string
	cwd, err := os.Getwd()
	if err != nil {
		return env, err
	}
	envFilePath := stage1initcommon.EnvFilePath(cwd, types.ACName(appName))
	var envFile *os.File
	if envFile, err = os.Open(envFilePath); err != nil {
		return env, err
	}
	defer envFile.Close()
	scanner := bufio.NewScanner(envFile)
	for scanner.Scan() {
		env = append(env, scanner.Text())
	}
	err = scanner.Err()
	return env, err
}

func execArgs(envv []string) error {
	argv0 := flag.Arg(0)
	argv := flag.Args()

	return syscall.Exec(argv0, argv, envv)
}

func main() {
	flag.Parse()

	log.SetDebug(debug)
	diag.SetDebug(debug)

	if !debug {
		diag.SetOutput(ioutil.Discard)
	}

	root, err := getRootDir(podPid)
	if err != nil {
		log.FatalE("Failed to get pod root", err)
	}

	env, err := readEnv()
	if err != nil {
		log.FatalE("Failed to read app env", err)
	}

	if err := os.Chdir(root); err != nil {
		log.FatalE("Failed to change to new root", err)
	}

	if err := syscall.Chroot(root); err != nil {
		log.FatalE("Failed to chroot", err)
	}

	diag.Println("PID:", podPid)
	diag.Println("APP:", appName)
	diag.Println("ARGS:", flag.Args())

	if err := execArgs(env); err != nil {
		log.PrintE("exec failed", err)
	}

	os.Exit(254)
}
