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

package metadata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	simulator "github.com/rdimitrov/go-tuf-metadata/testutils/simulators"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {

	err := simulator.SetupTestDirs()
	defer simulator.Cleanup()

	if err != nil {
		log.Fatalf("failed to setup test dirs: %v", err)
	}
	m.Run()
}

func TestGenericRead(t *testing.T) {
	// Assert that it chokes correctly on an unknown metadata type
	badMetadata := "{\"signed\": {\"_type\": \"bad-metadata\"}}"
	_, err := Root().FromBytes([]byte(badMetadata))
	assert.ErrorContains(t, err, "expected metadata type root, got - bad-metadata")
	_, err = Snapshot().FromBytes([]byte(badMetadata))
	assert.ErrorContains(t, err, "expected metadata type snapshot, got - bad-metadata")
	_, err = Targets().FromBytes([]byte(badMetadata))
	assert.ErrorContains(t, err, "expected metadata type targets, got - bad-metadata")
	_, err = Timestamp().FromBytes([]byte(badMetadata))
	assert.ErrorContains(t, err, "expected metadata type timestamp, got - bad-metadata")

	badMetadataPath := fmt.Sprintf("%s/bad-metadata.json", simulator.RepoDir)
	err = simulator.CreateFile(badMetadataPath, []byte(badMetadata))
	assert.NoError(t, err)
	assert.FileExists(t, badMetadataPath)

	_, err = Root().FromFile(badMetadataPath)
	assert.ErrorContains(t, err, "expected metadata type root, got - bad-metadata")
	_, err = Snapshot().FromFile(badMetadataPath)
	assert.ErrorContains(t, err, "expected metadata type snapshot, got - bad-metadata")
	_, err = Targets().FromFile(badMetadataPath)
	assert.ErrorContains(t, err, "expected metadata type targets, got - bad-metadata")
	_, err = Timestamp().FromFile(badMetadataPath)
	assert.ErrorContains(t, err, "expected metadata type timestamp, got - bad-metadata")

	err = simulator.DeleteFile(badMetadataPath)
	assert.NoError(t, err)
	assert.NoFileExists(t, badMetadataPath)
}

func TestMDReadWriteFileExceptions(t *testing.T) {
	// Test writing to a file with bad filename
	badMetadataPath := fmt.Sprintf("%s/bad-metadata.json", simulator.RepoDir)
	_, err := Root().FromFile(badMetadataPath)
	assert.ErrorContains(t, err, fmt.Sprintf("open %s: no such file or directory", badMetadataPath))

	// Test serializing to a file with bad filename
	root, err := Root().FromFile(fmt.Sprintf("%s/root.json", simulator.RepoDir))
	assert.NoError(t, err)
	err = root.ToFile("", false)
	assert.ErrorContains(t, err, "no such file or directory")
}

func TestRootReadWriteReadCompare(t *testing.T) {
	path1 := simulator.RepoDir + "/root.json"
	root1, err := Root().FromFile(path1)
	assert.NoError(t, err)

	path2 := path1 + ".tmp"
	err = root1.ToFile(path2, false)
	assert.NoError(t, err)

	root2, err := Root().FromFile(path2)
	assert.NoError(t, err)

	bytes1, err := root1.ToBytes(false)
	assert.NoError(t, err)
	bytes2, err := root2.ToBytes(false)
	assert.NoError(t, err)
	assert.Equal(t, bytes1, bytes2)

	err = simulator.DeleteFile(path2)
	assert.NoError(t, err)
}

func TestSnapshotReadWriteReadCompare(t *testing.T) {
	path1 := simulator.RepoDir + "/snapshot.json"
	snaphot1, err := Snapshot().FromFile(path1)
	assert.NoError(t, err)

	path2 := path1 + ".tmp"
	err = snaphot1.ToFile(path2, false)
	assert.NoError(t, err)

	snapshot2, err := Snapshot().FromFile(path2)
	assert.NoError(t, err)

	bytes1, err := snaphot1.ToBytes(false)
	assert.NoError(t, err)
	bytes2, err := snapshot2.ToBytes(false)
	assert.NoError(t, err)
	assert.Equal(t, bytes1, bytes2)

	err = simulator.DeleteFile(path2)
	assert.NoError(t, err)
}

func TestTargetsReadWriteReadCompare(t *testing.T) {
	path1 := simulator.RepoDir + "/targets.json"
	targets1, err := Targets().FromFile(path1)
	assert.NoError(t, err)

	path2 := path1 + ".tmp"
	err = targets1.ToFile(path2, false)
	assert.NoError(t, err)

	targets2, err := Targets().FromFile(path2)
	assert.NoError(t, err)

	bytes1, err := targets1.ToBytes(false)
	assert.NoError(t, err)
	bytes2, err := targets2.ToBytes(false)
	assert.NoError(t, err)
	assert.Equal(t, bytes1, bytes2)

	err = simulator.DeleteFile(path2)
	assert.NoError(t, err)
}

func TestTimestampReadWriteReadCompare(t *testing.T) {
	path1 := simulator.RepoDir + "/timestamp.json"
	timestamp1, err := Timestamp().FromFile(path1)
	assert.NoError(t, err)

	path2 := path1 + ".tmp"
	err = timestamp1.ToFile(path2, false)
	assert.NoError(t, err)

	timestamp2, err := Timestamp().FromFile(path2)
	assert.NoError(t, err)

	bytes1, err := timestamp1.ToBytes(false)
	assert.NoError(t, err)
	bytes2, err := timestamp2.ToBytes(false)
	assert.NoError(t, err)
	assert.Equal(t, bytes1, bytes2)

	err = simulator.DeleteFile(path2)
	assert.NoError(t, err)
}

func TestSerializeAndValidate(t *testing.T) {
	// Assert that by changing one required attribute validation will fail.
	root, err := Root().FromFile(simulator.RepoDir + "/root.json")
	assert.NoError(t, err)
	root.Signed.Version = 0

	_, err = root.ToBytes(false)
	// TODO: refering to python-tuf, this should fail
	assert.NoError(t, err)
}

func TrimBytes(data []byte) ([]byte, error) {
	buffer := new(bytes.Buffer)
	err := json.Compact(buffer, data)
	if err != nil {
		log.Debugf("failed to trim bytes: %v", err)
		return data, err
	}
	data = buffer.Bytes()
	return data, nil
}

func TestToFromBytes(t *testing.T) {
	// ROOT
	data, err := simulator.ReadFile(simulator.RepoDir + "/root.json")
	assert.NoError(t, err)
	data, err = TrimBytes(data)
	assert.NoError(t, err)
	root, err := Root().FromBytes(data)
	assert.NoError(t, err)

	// Comparate that from_bytes/to_bytes doesn't change the content
	// for two cases for the serializer: noncompact and compact.

	// Case 1: test noncompact by overriding the default serializer.
	rootBytes, err := root.ToBytes(false)
	assert.NoError(t, err)
	assert.Equal(t, data, rootBytes)

	// Case 2: test compact by using the default serializer.
	root2, err := Root().FromBytes(rootBytes)
	assert.NoError(t, err)
	root2Bytes, err := root2.ToBytes(false)
	assert.NoError(t, err)
	assert.Equal(t, rootBytes, root2Bytes)

	// SNAPSHOT
	data, err = simulator.ReadFile(simulator.RepoDir + "/snapshot.json")
	assert.NoError(t, err)
	data, err = TrimBytes(data)
	assert.NoError(t, err)
	snapshot, err := Snapshot().FromBytes(data)
	assert.NoError(t, err)

	// Case 1: test noncompact by overriding the default serializer.
	snapshotBytes, err := snapshot.ToBytes(false)
	assert.NoError(t, err)
	assert.Equal(t, string(data), string(snapshotBytes))

	// Case 2: test compact by using the default serializer.
	snapshot2, err := Snapshot().FromBytes(snapshotBytes)
	assert.NoError(t, err)
	snapshot2Bytes, err := snapshot2.ToBytes(false)
	assert.NoError(t, err)
	assert.Equal(t, string(snapshotBytes), string(snapshot2Bytes))

	// TARGETS
	data, err = simulator.ReadFile(simulator.RepoDir + "/targets.json")
	assert.NoError(t, err)
	data, err = TrimBytes(data)
	assert.NoError(t, err)
	targets, err := Targets().FromBytes(data)
	assert.NoError(t, err)

	// Case 1: test noncompact by overriding the default serializer.
	targetsBytes, err := targets.ToBytes(false)
	assert.NoError(t, err)
	assert.Equal(t, string(data), string(targetsBytes))

	// Case 2: test compact by using the default serializer.
	targets2, err := Targets().FromBytes(targetsBytes)
	assert.NoError(t, err)
	targets2Bytes, err := targets2.ToBytes(false)
	assert.NoError(t, err)
	assert.Equal(t, string(targetsBytes), string(targets2Bytes))

	// TIMESTAMP
	data, err = simulator.ReadFile(simulator.RepoDir + "/timestamp.json")
	assert.NoError(t, err)
	data, err = TrimBytes(data)
	assert.NoError(t, err)
	timestamp, err := Timestamp().FromBytes(data)
	assert.NoError(t, err)

	// Case 1: test noncompact by overriding the default serializer.
	timestampBytes, err := timestamp.ToBytes(false)
	assert.NoError(t, err)
	assert.Equal(t, string(data), string(timestampBytes))

	// Case 2: test compact by using the default serializer.
	timestamp2, err := Timestamp().FromBytes(timestampBytes)
	assert.NoError(t, err)
	timestamp2Bytes, err := timestamp2.ToBytes(false)
	assert.NoError(t, err)
	assert.Equal(t, string(timestampBytes), string(timestamp2Bytes))

}
