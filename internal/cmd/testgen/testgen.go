// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// testgen downloads the current testsuite from json-schema-org
// and updates the tests in the tree.
// This is normally invoked by "go generate" in the internal/tests directory.
package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	tempDir, err := os.MkdirTemp("", "jsonschema-testgen")
	if err != nil {
		log.Fatal(err)
	}

	const repo = "https://github.com/json-schema-org/JSON-Schema-Test-Suite"
	cmd := exec.Command("git", "clone", repo)
	cmd.Dir = tempDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("> git clone %s\n", repo)
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

	for _, dir := range []string{"output-tests", "remotes", "tests"} {
		copyDir(filepath.Join(tempDir, "JSON-Schema-Test-Suite", dir), dir)
		copyFile(filepath.Join(tempDir, "JSON-Schema-Test-Suite", "LICENSE"), filepath.Join(dir, "LICENSE"))
		writeREADME(filepath.Join(dir, "README"))
	}
}

// copyDir copies all the files in a directory, recursively.
func copyDir(fromDir, toDir string) {
	fsys := os.DirFS(fromDir)
	keep := make(map[string]bool)
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if strings.HasPrefix(path, "README") {
			return nil
		}

		fpath, err := filepath.Localize(path)
		if err != nil {
			log.Fatal(err)
		}
		newPath := filepath.Join(toDir, fpath)
		if d.IsDir() {
			if err := os.MkdirAll(newPath, 0o755); err != nil {
				log.Fatal(err)
			}
			keep[fpath] = true
			return nil
		}

		if d.Type()&fs.ModeSymlink != 0 {
			// The testsuite has a "latest" symlink.
			// Ignore it.
			return nil
		}
		if !d.Type().IsRegular() {
			log.Fatalf("%s: not a regular file: mode %s %#o", path, d.Type(), d.Type())
		}

		info, err := fs.Stat(fsys, path)
		if err != nil {
			log.Fatal(err)
		}
		if info.Mode()&0o111 != 0 {
			log.Fatalf("%s is executable: mode %s %#o", path, info.Mode(), info.Mode())
		}

		newContents, err := fs.ReadFile(fsys, path)
		if err != nil {
			log.Fatal(err)
		}

		oldContents, err := os.ReadFile(newPath)
		if err == nil && bytes.Equal(newContents, oldContents) {
			// File is unchanged.
			keep[fpath] = true
			return nil
		}

		fmt.Printf("updating %s\n", path)

		if err := os.WriteFile(newPath, newContents, 0o644); err != nil {
			log.Fatal(err)
		}

		keep[fpath] = true

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	err = fs.WalkDir(os.DirFS(toDir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if keep[path] {
			return nil
		}

		if path != "LICENSE" && path != "README" {
			fmt.Printf("removing %s\n", path)
		}

		if err := os.RemoveAll(filepath.Join(toDir, path)); err != nil {
			log.Fatal(err)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

// copyFile copies a file.
func copyFile(fromFile, toFile string) {
	data, err := os.ReadFile(fromFile)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(toFile, data, 0o644); err != nil {
		log.Fatal(err)
	}
}

// writeREADME writes our local README file.
func writeREADME(to string) {
	if err := os.WriteFile(to, []byte(readme), 0o644); err != nil {
		log.Fatal(err)
	}
}

const readme = `The contents of this directory are copied from
https://github.com/json-schema-org/JSON-Schema-Test-Suite.

The files in this directory are covered by the LICENSE file
in this directory. These files are only used for testing,
and the LICENSE only applies to explicit copies of these files.
Packages that import the jsonschema packages will not import
these files, and will not be subject to this LICENSE.
`
