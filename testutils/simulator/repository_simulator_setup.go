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
	"os"
	"time"

	"github.com/rdimitrov/go-tuf-metadata/metadata"
)

var (
	MetadataURL = "https://jku.github.io/tuf-demo/metadata"
	TargetsURL  = "https://jku.github.io/tuf-demo/targets"

	MetadataDir  string
	RootBytes    []byte
	PastDateTime time.Time
	Sim          *RepositorySimulator

	metadataPath = "/metadata"
	targetsPath  = "/targets"
	LocalDir     string
	DumpDir      string
)

func InitLocalEnv() error {
	log := metadata.GetLogger()

	tmp := os.TempDir()

	tmpDir, err := os.MkdirTemp(tmp, "0750")
	if err != nil {
		log.Error(err, "failed to create temporary directory")
	}

	os.Mkdir(tmpDir+metadataPath, 0750)
	os.Mkdir(tmpDir+targetsPath, 0750)
	LocalDir = tmpDir
	return nil
}

func InitMetadataDir() (*RepositorySimulator, string, string, error) {
	log := metadata.GetLogger()

	err := InitLocalEnv()
	if err != nil {
		log.Error(err, "failed to initialize environment")
	}
	metadataDir := LocalDir + metadataPath

	sim := NewRepository()

	f, err := os.Create(metadataDir + "/root.json")
	if err != nil {
		log.Error(err, "failed to create root")
	}

	f.Write(sim.SignedRoots[0])
	targetsDir := LocalDir + targetsPath
	sim.LocalDir = LocalDir
	return sim, metadataDir, targetsDir, err
}

func GetRootBytes(localMetadataDir string) ([]byte, error) {
	return os.ReadFile(localMetadataDir + "/root.json")
}

func RepositoryCleanup(tmpDir string) {
	log := metadata.GetLogger()

	log.V(4).Info("Cleaning temporary directory", tmpDir)
	os.RemoveAll(tmpDir)
}
