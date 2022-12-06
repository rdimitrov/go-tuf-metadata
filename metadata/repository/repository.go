package repository

import (
	"github.com/rdimitrov/ngo-tuf/metadata"
)

// repositoryType struct for storing metadata
type repositoryType struct {
	root      *metadata.Metadata[metadata.RootType]
	snapshot  *metadata.Metadata[metadata.SnapshotType]
	timestamp *metadata.Metadata[metadata.TimestampType]
	targets   map[string]*metadata.Metadata[metadata.TargetsType]
}

// New creates an empty repository instance
func New() *repositoryType {
	return &repositoryType{
		targets: map[string]*metadata.Metadata[metadata.TargetsType]{},
	}
}

// Root returns metadata of type Root
func (r *repositoryType) Root() *metadata.Metadata[metadata.RootType] {
	return r.root
}

// SetRoot sets metadata of type Root
func (r *repositoryType) SetRoot(meta *metadata.Metadata[metadata.RootType]) {
	r.root = meta
}

// Snapshot returns metadata of type Snapshot
func (r *repositoryType) Snapshot() *metadata.Metadata[metadata.SnapshotType] {
	return r.snapshot
}

// SetSnapshot sets metadata of type Snapshot
func (r *repositoryType) SetSnapshot(meta *metadata.Metadata[metadata.SnapshotType]) {
	r.snapshot = meta
}

// Timestamp returns metadata of type Timestamp
func (r *repositoryType) Timestamp() *metadata.Metadata[metadata.TimestampType] {
	return r.timestamp
}

// SetTimestamp sets metadata of type Timestamp
func (r *repositoryType) SetTimestamp(meta *metadata.Metadata[metadata.TimestampType]) {
	r.timestamp = meta
}

// Targets returns metadata of type Targets
func (r *repositoryType) Targets(name string) *metadata.Metadata[metadata.TargetsType] {
	return r.targets[name]
}

// SetTargets sets metadata of type Targets
func (r *repositoryType) SetTargets(name string, meta *metadata.Metadata[metadata.TargetsType]) {
	r.targets[name] = meta
}