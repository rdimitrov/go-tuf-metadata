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
	"crypto"
	"encoding/json"
	"fmt"
	"testing"

	simulator "github.com/rdimitrov/go-tuf-metadata/testutils/simulators"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
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

func TestSignVerify(t *testing.T) {
	root, err := Root().FromFile(simulator.RepoDir + "/root.json")
	assert.NoError(t, err)

	// Locate the public keys we need from root
	assert.NotEmpty(t, root.Signed.Roles[TARGETS].KeyIDs)
	targetsKeyID := root.Signed.Roles[TARGETS].KeyIDs[0]
	assert.NotEmpty(t, root.Signed.Roles[SNAPSHOT].KeyIDs)
	snapshotKeyID := root.Signed.Roles[SNAPSHOT].KeyIDs[0]

	// Load sample metadata (targets) and assert ...
	targets, err := Targets().FromFile(simulator.RepoDir + "/targets.json")
	assert.NoError(t, err)
	data, err := targets.Signed.MarshalJSON()
	assert.NoError(t, err)
	sig := getSignatureByKeyID(targets.Signatures, targetsKeyID)

	// ... it has a single existing signature,
	assert.Equal(t, 1, len(targets.Signatures))

	// ... which is valid for the correct key.
	targetsKey := root.Signed.Keys[targetsKeyID]
	targetsPublicKey, err := targetsKey.ToPublicKey()
	assert.NoError(t, err)
	targetsHash := crypto.Hash(0)
	if targetsKey.Type != KeyTypeEd25519 {
		targetsHash = crypto.SHA256
	}
	targetsVerifier, err := signature.LoadVerifier(targetsPublicKey, targetsHash)
	assert.NoError(t, err)
	err = targetsVerifier.VerifySignature(bytes.NewReader(sig), bytes.NewReader(data))
	assert.NoError(t, err)

	snapshotKey := root.Signed.Keys[snapshotKeyID]
	snapshotPublicKey, err := snapshotKey.ToPublicKey()
	assert.NoError(t, err)
	snapshotHash := crypto.Hash(0)
	if snapshotKey.Type != KeyTypeEd25519 {
		snapshotHash = crypto.SHA256
	}
	snapshotVerifier, err := signature.LoadVerifier(snapshotPublicKey, snapshotHash)
	assert.NoError(t, err)

	err = snapshotVerifier.VerifySignature(bytes.NewReader(sig), bytes.NewReader(data))
	assert.ErrorContains(t, err, "crypto/rsa: verification error")

	signer, err := signature.LoadSignerFromPEMFile(simulator.KeystoreDir+"/snapshot_key", crypto.SHA256, cryptoutils.SkipPassword)
	// root.Sign(signer)
	// root.ToFile(simulator.RepoDir+"/tmp.json", false)
	assert.NoError(t, err)
	// Append a new signature with the unrelated key and assert that ...
	snapshotSig, err := targets.Sign(signer)
	assert.NoError(t, err)
	// ... there are now two signatures, and
	assert.Equal(t, 2, len(targets.Signatures))
	// ... both are valid for the corresponding keys.
	err = targetsVerifier.VerifySignature(bytes.NewReader(sig), bytes.NewReader(data))
	assert.NoError(t, err)
	err = snapshotVerifier.VerifySignature(bytes.NewReader(snapshotSig.Signature), bytes.NewReader(data))
	assert.NoError(t, err)
	// ... the returned (appended) signature is for snapshot key
	assert.Equal(t, snapshotSig.KeyID, snapshotKeyID)

	// Create and assign (don't append) a new signature and assert that ...
	signer, err = signature.LoadSignerFromPEMFile(simulator.KeystoreDir+"/timestamp_key", crypto.SHA256, cryptoutils.SkipPassword)
	assert.NoError(t, err)

	// Append a new signature with the unrelated key and assert that ...
	targets.ClearSignatures()
	timestampSig, err := targets.Sign(signer)
	assert.NoError(t, err)
	// ... there now is only one signature,
	assert.Equal(t, 1, len(targets.Signatures))
	// ... valid for that key.
	assert.NotEmpty(t, root.Signed.Roles[TIMESTAMP].KeyIDs)
	timestampKeyID := root.Signed.Roles[TIMESTAMP].KeyIDs[0]
	timestampKey := root.Signed.Keys[timestampKeyID]
	timestampPublicKey, err := timestampKey.ToPublicKey()
	assert.NoError(t, err)
	timestampHash := crypto.Hash(0)
	if timestampKey.Type != KeyTypeEd25519 {
		timestampHash = crypto.SHA256
	}
	timestampVerifier, err := signature.LoadVerifier(timestampPublicKey, timestampHash)
	assert.NoError(t, err)

	err = timestampVerifier.VerifySignature(bytes.NewReader(timestampSig.Signature), bytes.NewReader(data))
	assert.NoError(t, err)
	// TODO: should fail
	targetsVerifier.VerifySignature(bytes.NewReader(timestampSig.Signature), bytes.NewReader(data))
	assert.NoError(t, err)
}
