// ## Build
//
// The build process is quite straightforward: we first recreate the `dist`
// folder that contains artifacts and loop over the defined lambda
// functions to compile and zip the resulting executable to the `dist`
// folder.
package build

import (
	"github.com/pkg/errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

//
// Currently, we have 2 lambda functions creating custom resources:
// `dnscert` and `loggrp`. For more information, consult the
// code of each functions.
//
var Functions = []string{"cogclientset", "dnscert", "loggrp"}

func Build() error {
	for _, function := range Functions {
		dir, err := FunctionDir(function)
		if err != nil {
			return err
		}
		if err = compileFunction(dir); err != nil {
			return err
		}
	}
	return nil
}

func FunctionDir(function string) (string, error) {
	dir, err := filepath.Abs("../" + function)
	if err != nil {
		log.Fatal(err)
	}
	return dir, nil
}

func compileFunction(dir string) error {
	// TODO@stan: find a way to call `go build` directly (with cross compile).
	path, err := filepath.Abs("./buildFunction.sh")
	if err != nil {
		return errors.Wrap(err, "cannot find the script buildFunction.sh")
	}
	cmd := &exec.Cmd{
		Path: path,
		Args: []string{"buildFunction.sh"},
	}
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "cannot call the script buildFunction.sh")
	}
	return nil
}
