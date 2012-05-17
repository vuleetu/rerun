package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/build"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"
)

func install(buildpath, lastError string) (installed bool, errorOutput string, err error) {
	cmdline := []string{"go", "install", "-v", buildpath}

	cmd := exec.Command("go", cmdline[1:]...)
	bufOut := bytes.NewBuffer([]byte{})
	bufErr := bytes.NewBuffer([]byte{})
	cmd.Stdout = bufOut
	cmd.Stderr = bufErr

	err = cmd.Run()

	if bufOut.Len() != 0 {
		errorOutput = bufOut.String()
		if errorOutput != lastError {
			fmt.Print(bufOut)
		}
		err = errors.New("compile error")
		return
	}

	installed = bufErr.Len() != 0

	return
}

func run(binName, binPath string, args []string) (runch chan bool) {
	runch = make(chan bool)
	go func() {
		cmdline := append([]string{binName}, args...)
		var proc *os.Process
		for _ = range runch {
			if proc != nil {
				proc.Kill()
			}
			cmd := exec.Command(binPath, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			log.Print(cmdline)
			cmd.Start()
			proc = cmd.Process
		}
	}()
	return
}

func rerun(buildpath string, args []string) (err error) {
	log.Printf("setting up rerun for %s %v", buildpath, args)

	pkg, err := build.Import(buildpath, "", 0)
	if err != nil {
		return
	}

	if pkg.Name != "main" {
		err = errors.New(fmt.Sprintf("expected package %q, got %q", "main", pkg.Name))
	}

	_, binName := path.Split(buildpath)
	binPath := filepath.Join(pkg.BinDir, binName)

	runch := run(binName, binPath, args)

	var errorOutput string
	_, errorOutput, ierr := install(buildpath, errorOutput)
	if ierr == nil {
		runch <- true
	}

	for {
		var installed bool
		installed, errorOutput, _ = install(buildpath, errorOutput)
		if installed {
			runch <- true
		}
		time.Sleep(1e9)
	}
	return
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: rerun <import path> [arg]*")
	}
	err := rerun(os.Args[1], os.Args[2:])
	if err != nil {
		log.Print(err)
	}
}