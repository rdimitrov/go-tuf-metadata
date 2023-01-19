// Copyright 2022-2023 VMware, Inc.
//
// This product is licensed to you under the BSD-2 license (the "License").
// You may not use this product except in compliance with the BSD-2 License.
// This product may include a number of subcomponents with separate copyright
// notices and license terms. Your use of these subcomponents is subject to
// the terms and conditions of the subcomponent's license, as noted in the
// LICENSE file.
//
// SPDX-License-Identifier: BSD-2-Clause

package updater

import (
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rdimitrov/go-tuf-metadata/metadata"
	"github.com/rdimitrov/go-tuf-metadata/metadata/config"
	"github.com/rdimitrov/go-tuf-metadata/metadata/fetcher"
	"github.com/rdimitrov/go-tuf-metadata/metadata/trustedmetadata"
	log "github.com/sirupsen/logrus"
)

// Client update workflow implementation
// The "Updater" provides an implementation of the `TUF client workflow
// <https://theupdateframework.github.io/specification/latest/#detailed-client-workflow>`_.
// "Updater" provides an API to query available targets and to download them in a
// secure manner: All downloaded files are verified by signed metadata.
// High-level description of "Updater" functionality:
//   - Initializing an "Updater" loads and validates the trusted local root
//     metadata: This root metadata is used as the source of trust for all other
//     metadata.
//   - "Refresh()"" can optionally be called to update and load all top-level
//     metadata as described in the specification, using both locally cached
//     metadata and metadata downloaded from the remote repository. If refresh is
//     not done explicitly, it will happen automatically during the first target
//     info lookup.
//   - "Updater" can be used to download targets. For each target:
//   - "GetTargetInfo()"" is first used to find information about a
//     specific target. This will load new targets metadata as needed (from
//     local cache or remote repository).
//   - "FindCachedTarget()"" can optionally be used to check if a
//     target file is already locally cached.
//   - "DownloadTarget()" downloads a target file and ensures it is
//     verified correct by the metadata.
type roleParentTuple struct {
	Role   string
	Parent string
}

type Updater struct {
	metadataDir     string
	metadataBaseUrl string
	targetDir       string
	targetBaseUrl   string
	trusted         *trustedmetadata.TrustedMetadata
	config          *config.UpdaterConfig
	fetcher         fetcher.Fetcher
}

// New creates a new “Updater“ instance and loads trusted root metadata.
func New(metadataDir, metadataBaseUrl, targetDir, targetBaseUrl string, f fetcher.Fetcher) (*Updater, error) {
	// use the built-in download fetcher if nothing is provided
	if f == nil {
		f = &fetcher.DefaultFetcher{}
	}
	// create an updater instance
	updater := &Updater{
		metadataDir:     metadataDir,
		metadataBaseUrl: ensureTrailingSlash(metadataBaseUrl),
		targetDir:       targetDir,
		targetBaseUrl:   ensureTrailingSlash(targetBaseUrl),
		config:          config.New(),
		fetcher:         f,
	}
	// load the root metadata file used for bootstrapping trust
	rootBytes, err := updater.loadLocalMetadata(metadata.ROOT)
	if err != nil {
		return nil, err
	}
	// create a new trusted metadata instance
	trustedMetadataSet, err := trustedmetadata.New(rootBytes)
	if err != nil {
		return nil, err
	}
	updater.trusted = trustedMetadataSet
	return updater, nil
}

// Refresh refreshes top-level metadata.
// Downloads, verifies, and loads metadata for the top-level roles in the
// specified order (root -> timestamp -> snapshot -> targets) implementing
// all the checks required in the TUF client workflow.
// A Refresh()“ can be done only once during the lifetime of an Updater.
// If Refresh()“ has not been explicitly called before the first
// “GetTargetInfo()“ call, it will be done implicitly at that time.
// The metadata for delegated roles is not updated by Refresh()“:
// that happens on demand during GetTargetInfo()“. However, if the
// repository uses `consistent_snapshot
// <https://theupdateframework.github.io/specification/latest/#consistent-snapshots>`_,
// then all metadata downloaded by the Updater will use the same consistent
// repository state.
func (update *Updater) Refresh() error {
	err := update.loadRoot()
	if err != nil {
		return err
	}
	err = update.loadTimestamp()
	if err != nil {
		return err
	}
	err = update.loadSnapshot()
	if err != nil {
		return err
	}
	_, err = update.loadTargets(metadata.TARGETS, metadata.ROOT)
	if err != nil {
		return err
	}
	return nil
}

// GetTargetInfo returns “metadata.TargetFiles“ instance with information
// for targetPath. The return value can be used as an argument to
// “DownloadTarget()“ and “FindCachedTarget()“.
// If “Refresh()“ has not been called before calling
// “GetTargetInfo()“, the refresh will be done implicitly.
// As a side-effect this method downloads all the additional (delegated
// targets) metadata it needs to return the target information.
func (update *Updater) GetTargetInfo(targetPath string) (*metadata.TargetFiles, error) {
	// do a Refresh() in case there's no trusted targets.json yet
	if update.trusted.Targets[metadata.TARGETS] == nil {
		err := update.Refresh()
		if err != nil {
			return nil, err
		}
	}
	return update.preOrderDepthFirstWalk(targetPath)
}

// DownloadTarget downloads the target file specified by targetFile
func (update *Updater) DownloadTarget(targetFile *metadata.TargetFiles, filePath, targetBaseURL string) (string, error) {
	var err error
	if filePath == "" {
		filePath, err = update.generateTargetFilePath(targetFile)
		if err != nil {
			return "", err
		}
	}
	if targetBaseURL == "" {
		if update.targetBaseUrl == "" {
			return "", fmt.Errorf("targetBaseURL must be set in either DownloadTarget() or the Updater struct")
		}
		targetBaseURL = update.targetBaseUrl
	} else {
		targetBaseURL = ensureTrailingSlash(targetBaseURL)
	}
	targetFilePath := targetFile.Path
	consistentSnapshot := update.trusted.Root.Signed.ConsistentSnapshot
	if consistentSnapshot && update.config.PrefixTargetsWithHash {
		hashes := ""
		// get first hex value of hashes
		for _, v := range targetFile.Hashes {
			hashes = hex.EncodeToString(v)
			break
		}
		dirName, baseName, ok := strings.Cut(targetFilePath, "/")
		if !ok {
			return "", fmt.Errorf("error handling targetFilePath %s, no separator found", targetFilePath)
		}
		targetFilePath = fmt.Sprintf("%s/%s.%s", dirName, hashes, baseName)
	}
	fullURL := fmt.Sprintf("%s%s", targetBaseURL, targetFilePath)
	data, err := update.fetcher.DownloadFile(fullURL, targetFile.Length)
	if err != nil {
		return "", err
	}
	err = targetFile.VerifyLengthHashes(data)
	if err != nil {
		return "", err
	}
	// write the data content to the temporary file
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return "", err
	}
	log.Debugf("Downloaded target %s", targetFile.Path)
	return filePath, nil
}

// FindCachedTarget checks whether a local file is an up to date target
func (update *Updater) FindCachedTarget(targetFile *metadata.TargetFiles, filePath string) (string, error) {
	var err error
	targetFilePath := ""
	// get its path if not provided
	if filePath == "" {
		targetFilePath, err = update.generateTargetFilePath(targetFile)
		if err != nil {
			return "", err
		}
	} else {
		targetFilePath = filePath
	}
	// get file content
	data, err := readFile(targetFilePath)
	if err != nil {
		return "", err
	}
	// verify if this local target file is an up-to-date target
	err = targetFile.VerifyLengthHashes(data)
	if err != nil {
		return "", err
	}
	// if all okay, return its path
	return targetFilePath, nil
}

// loadTimestamp load local and remote timestamp metadata
func (update *Updater) loadTimestamp() error {
	// try to read local timestamp
	data, err := update.loadLocalMetadata(metadata.TIMESTAMP)
	if err == nil {
		// try to verify and load it to the trusted metadata set
		result, err := update.trusted.UpdateTimestamp(data)
		if result == nil && err != nil {
			return err
		}
		// proceed if both res and temp are != nil
		log.Debugf("local timestamp is verified, but expired: %s", err)
	}
	// Load from remote (whether local load succeeded or not)
	data, err = update.downloadMetadata(metadata.TIMESTAMP, update.config.TimestampMaxLength, "")
	if err != nil {
		return err
	}
	// if the new timestamp version is the same as current, discard the
	// new timestamp. This is normal and it shouldn't raise any error.
	// a bit hacky, but if both result and err are nil it means new timestamp
	// version is the same as current
	result, err := update.trusted.UpdateTimestamp(data)
	if result == nil && err != nil {
		return err
	}
	err = update.persistMetadata(metadata.TIMESTAMP, data)
	if err != nil {
		return err
	}
	return nil
}

// loadSnapshot load local (and if needed remote) snapshot metadata
func (update *Updater) loadSnapshot() error {
	// try to read local snapshot
	data, err := update.loadLocalMetadata(metadata.SNAPSHOT)
	if err == nil {
		// try to verify and load local snapshot to trusted metadata set
		_, err = update.trusted.UpdateSnapshot(data, true)
		if err == nil {
			log.Debugf("Local snapshot is valid: not downloading new one")
			return nil
		}
	}
	// local snapshot does not exist or is invalid, update from remote
	if update.trusted.Timestamp == nil {
		return fmt.Errorf("trusted timestamp not set")
	}
	snapshotMeta := update.trusted.Timestamp.Signed.Meta[fmt.Sprintf("%s.json", metadata.SNAPSHOT)]
	length := snapshotMeta.Length
	if length == 0 {
		length = update.config.SnapshotMaxLength
	}
	version := ""
	if update.trusted.Root.Signed.ConsistentSnapshot {
		version = strconv.FormatInt(snapshotMeta.Version, 10)
	}
	data, err = update.downloadMetadata(metadata.SNAPSHOT, length, version)
	if err != nil {
		return err
	}
	_, err = update.trusted.UpdateSnapshot(data, false)
	if err != nil {
		return err
	}
	err = update.persistMetadata(metadata.TIMESTAMP, data)
	if err != nil {
		return err
	}
	return nil
}

// loadTargets load local (and if needed remote) metadata for roleName.
func (update *Updater) loadTargets(roleName, parentName string) (*metadata.Metadata[metadata.TargetsType], error) {
	// avoid loading 'roleName' more than once during "GetTargetInfo"
	role, ok := update.trusted.Targets[roleName]
	if ok {
		return role, nil
	}
	// try to read local targets
	data, err := update.loadLocalMetadata(roleName)
	if err == nil {
		// try to verify and load local targets file to trusted metadata set
		delegatedTargets, err := update.trusted.UpdateDelegatedTargets(data, roleName, parentName)
		if err == nil {
			log.Debugf("Local %s is valid: not downloading new one\n", roleName)
			return delegatedTargets, nil
		}
	}
	// local 'role' does not exist or is invalid, update from remote
	if update.trusted.Snapshot == nil {
		return nil, fmt.Errorf("trusted snapshot not set")
	}
	metaInfo := update.trusted.Snapshot.Signed.Meta[fmt.Sprintf("%s.json", roleName)]
	length := metaInfo.Length
	if length != 0 {
		length = update.config.TargetsMaxLength
	}
	version := ""
	if update.trusted.Root.Signed.ConsistentSnapshot {
		version = strconv.FormatInt(metaInfo.Version, 10)
	}
	data, err = update.downloadMetadata(roleName, length, version)
	if err != nil {
		return nil, err
	}
	delegatedTargets, err := update.trusted.UpdateDelegatedTargets(data, roleName, parentName)
	if err != nil {
		return nil, err
	}
	err = update.persistMetadata(roleName, data)
	if err != nil {
		return nil, err
	}
	return delegatedTargets, nil
}

// loadRoot load remote root metadata. Sequentially load and
// persist on local disk every newer root metadata version
// available on the remote.
func (update *Updater) loadRoot() error {
	// calculate boundaries
	lowerBound := update.trusted.Root.Signed.Version + 1
	upperBound := lowerBound + update.config.MaxRootRotations

	// loop until we find the latest available version of root
	for nextVersion := lowerBound; nextVersion <= upperBound; nextVersion++ {
		data, err := update.downloadMetadata(metadata.ROOT, update.config.RootMaxLength, strconv.FormatInt(nextVersion, 10))
		if err != nil {
			// current root is newest available
			break
		}
		// verify and load the root data
		_, err = update.trusted.UpdateRoot(data)
		if err != nil {
			return err
		}
		// write root to disk
		err = update.persistMetadata(metadata.ROOT, data)
		if err != nil {
			return err
		}
	}
	return nil
}

// preOrderDepthFirstWalk interrogates the tree of target delegations
// in order of appearance (which implicitly order trustworthiness),
// and returns the matching target found in the most trusted role.
func (update *Updater) preOrderDepthFirstWalk(targetFilePath string) (*metadata.TargetFiles, error) {
	// list of delegations to be interrogated. A (role, parent role) pair
	// is needed to load and verify the delegated targets metadata.
	delegationsToVisit := []roleParentTuple{{
		Role:   metadata.TARGETS,
		Parent: metadata.ROOT,
	}}
	visitedRoleNames := map[string]bool{}
	// pre-order depth-first traversal of the graph of target delegations.
	for len(visitedRoleNames) <= update.config.MaxDelegations && len(delegationsToVisit) > 0 {
		// pop the role name from the top of the stack.
		delegation := delegationsToVisit[len(delegationsToVisit)-1]
		delegationsToVisit = delegationsToVisit[:len(delegationsToVisit)-1]
		// skip any visited current role to prevent cycles.
		_, ok := visitedRoleNames[delegation.Role]
		if ok {
			log.Debugf("Skipping visited current role %s\n", delegation.Role)
			continue
		}
		// The metadata for 'delegation.Role' must be downloaded/updated before
		// its targets, delegations, and child roles can be inspected.
		targets, err := update.loadTargets(delegation.Role, delegation.Parent)
		if err != nil {
			return nil, err
		}
		target, ok := targets.Signed.Targets[targetFilePath]
		if ok {
			log.Debugf("Found target in current role %s\n", delegation.Role)
			return &target, nil
		}
		// After pre-order check, add current role to set of visited roles.
		visitedRoleNames[delegation.Role] = true
		if targets.Signed.Delegations != nil {
			childRolesToVisit := []roleParentTuple{}
			// NOTE: This may be a slow operation if there are many
			// delegated roles.
			roles := targets.Signed.Delegations.GetRolesForTarget(targetFilePath)
			for child, terminating := range roles {
				log.Debugf("Adding child role %s", child)
				childRolesToVisit = append(childRolesToVisit, roleParentTuple{Role: child, Parent: delegation.Role})
				if terminating {
					log.Debugf("Not backtracking to other roles")
				}
				delegationsToVisit = []roleParentTuple{}
				break
			}
			// Push 'childRolesToVisit' in reverse order of appearance
			// onto 'delegationsToVisit'.  Roles are popped from the end of
			// the list.
			reverseSlice(childRolesToVisit)
			delegationsToVisit = append(delegationsToVisit, childRolesToVisit...)
		}
	}
	if len(delegationsToVisit) > 0 {
		log.Debugf("%d roles left to visit, but allowed at most %d delegations",
			len(delegationsToVisit),
			update.config.MaxDelegations)
	}
	// If this point is reached then target is not found, return nil
	return nil, fmt.Errorf("target %s not found", targetFilePath)
}

// persistMetadata writes metadata to disk atomically to avoid data loss.
func (update *Updater) persistMetadata(roleName string, data []byte) error {
	fileName := filepath.Join(update.metadataDir, fmt.Sprintf("%s.json", url.QueryEscape(roleName)))
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	// create a temporary file
	file, err := os.CreateTemp(cwd, "tuf_tmp")
	if err != nil {
		return err
	}
	// write the data content to the temporary file
	err = os.WriteFile(file.Name(), data, 0644)
	if err != nil {
		// delete the temporary file if there was an error while writing
		os.Remove(file.Name())
		return err
	}
	// if all okay, rename the temporary file to the desired one
	err = os.Rename(file.Name(), fileName)
	if err != nil {
		return err
	}
	return nil
}

// downloadMetadata download a metadata file and return it as bytes
func (update *Updater) downloadMetadata(roleName string, length int64, version string) ([]byte, error) {
	urlPath := update.metadataBaseUrl
	// build urlPath
	if version == "" {
		urlPath = fmt.Sprintf("%s%s.json", urlPath, url.QueryEscape(roleName))
	} else {
		urlPath = fmt.Sprintf("%s%s.%s.json", urlPath, version, url.QueryEscape(roleName))
	}
	return update.fetcher.DownloadFile(urlPath, length)
}

// generateTargetFilePath generates path from TargetFiles
func (update *Updater) generateTargetFilePath(tf *metadata.TargetFiles) (string, error) {
	if update.targetDir == "" {
		return "", fmt.Errorf("target_dir must be set if filepath is not given")
	}
	// Use URL encoded target path as filename
	return url.JoinPath(update.targetDir, url.QueryEscape(tf.Path))
}

// loadLocalMetadata reads a local <roleName>.json file and returns its bytes
func (update *Updater) loadLocalMetadata(roleName string) ([]byte, error) {
	roleName = fmt.Sprintf("%s.json", url.QueryEscape(roleName))
	return readFile(roleName)
}

// ensureTrailingSlash ensures url ends with a slash
func ensureTrailingSlash(url string) string {
	if strings.HasSuffix(url, "/") {
		return url
	}
	return url + "/"
}

// reverseSlice reverses the elements in a generic type of slice
func reverseSlice[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// readFile reads the content of a file and return its bytes
func readFile(name string) ([]byte, error) {
	in, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer in.Close()
	data, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}
	return data, nil
}
