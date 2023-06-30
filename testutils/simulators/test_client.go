// Copyright 2023 VMware, Inc.
//
// This product is licensed to you under the BSD-2 license (the "License").
// You may not use this product except in compliance with the BSD-2 License.
// This product may include a number of subcomponents with separate copyright
// notices and license terms. Your use of these subcomponents is subject to
// the terms and conditions of the subcomponent's license, as noted in the
// LICENSE file.
//
// SPDX-License-Identifier: BSD-2-Clause

package simulator

import (
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

var (
	TempDir     string
	RepoDir     string
	KeystoreDir string
)

func SetupTestDirs() error {
	tmp := fs.TempDir()
	var err error
	TempDir, err = fs.MkdirTemp(tmp, "0750")
	if err != nil {
		log.Fatal("failed to create temporary directory: ", err)
		return err
	}

	RepoDir = fmt.Sprintf("%s/repository_data/repository", TempDir)
	absPath, err := filepath.Abs("../testutils/repository_data/repository/metadata")
	if err != nil {
		log.Debugf("failed to get absolute path: %v", err)
	}
	err = fs.Copy(absPath, RepoDir)
	if err != nil {
		log.Debugf("failed to copy metadata to %s: %v", RepoDir, err)
		return err
	}

	KeystoreDir = fmt.Sprintf("%s/keystore", TempDir)
	err = fs.Mkdir(KeystoreDir)
	if err != nil {
		log.Debugf("failed to create keystore dir %s: %v", KeystoreDir, err)
	}
	absPath, err = filepath.Abs("../testutils/repository_data/keystore")
	if err != nil {
		log.Debugf("failed to get absolute path: %v", err)
	}
	err = fs.Copy(absPath, KeystoreDir)
	if err != nil {
		log.Debugf("failed to copy keystore to %s: %v", KeystoreDir, err)
		return err
	}

	return nil
}

func Cleanup() {
	log.Printf("cleaning temporary directory: %s\n", TempDir)
	err := fs.RemoveAll(TempDir)
	if err != nil {
		log.Fatalf("failed to cleanup test directories: %v", err)
	}
}
