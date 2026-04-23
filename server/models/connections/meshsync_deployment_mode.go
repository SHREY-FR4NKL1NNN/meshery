// Package connections provides Meshery-local connection helpers that complement
// the canonical `connection.Connection` wire contract defined in
// github.com/meshery/schemas/models/v1beta3/connection.
//
// This file hosts the MeshsyncDeploymentMode helpers, previously declared in
// `github.com/meshery/schemas/models/v1beta1/connection/connection_helper.go`.
// The helpers are Meshery domain logic (how the server decides whether to run
// meshsync in operator vs. embedded mode against a Kubernetes connection) and
// do not belong to the wire contract that v1beta3 canonicalises. Moving them
// here lets the Phase 3 consumer repoint drop every `v1beta1/connection`
// import from meshery/meshery without waiting for a schemas-side port.
//
// TODO(identifier-naming-migration): Once schemas releases a v1beta3
// connection_helper with these helpers (or an equivalent package), switch back
// to the shared definition and delete this file.
package connections

import (
	"github.com/meshery/schemas/models/core"
)

// MeshsyncDeploymentMode is the deployment mode of the meshsync controller for
// a given Kubernetes connection. Values mirror the strings persisted in
// `Connection.Metadata[MeshsyncDeploymentModeMetadataKey]`.
type MeshsyncDeploymentMode string

// MeshsyncDeploymentModeMetadataKey is the key under which the deployment mode
// is stored on a connection's metadata map. Kept as snake_case because it is
// persisted on the wire alongside other metadata entries and renaming it
// would break every connection already carrying the key.
const MeshsyncDeploymentModeMetadataKey = "meshsync_deployment_mode"

const (
	MeshsyncDeploymentModeOperator  MeshsyncDeploymentMode = "operator"
	MeshsyncDeploymentModeEmbedded  MeshsyncDeploymentMode = "embedded"
	MeshsyncDeploymentModeUndefined MeshsyncDeploymentMode = "undefined"
	MeshsyncDeploymentModeDefault                          = MeshsyncDeploymentModeEmbedded
)

// MeshsyncDeploymentModeFromString coerces a free-form string (typically a
// config value or a metadata entry) into one of the known deployment modes.
// Anything that does not match a known value collapses to
// MeshsyncDeploymentModeUndefined so callers can apply their own fallback.
func MeshsyncDeploymentModeFromString(value string) MeshsyncDeploymentMode {
	switch value {
	case string(MeshsyncDeploymentModeOperator):
		return MeshsyncDeploymentModeOperator
	case string(MeshsyncDeploymentModeEmbedded):
		return MeshsyncDeploymentModeEmbedded
	default:
		return MeshsyncDeploymentModeUndefined
	}
}

// MeshsyncDeploymentModeFromMetadata extracts the deployment mode stored on
// a connection's metadata map. Both strongly-typed (MeshsyncDeploymentMode)
// and string-shaped values are accepted; any other type falls back to
// MeshsyncDeploymentModeUndefined.
func MeshsyncDeploymentModeFromMetadata(metadata core.Map) MeshsyncDeploymentMode {
	raw, exists := metadata[MeshsyncDeploymentModeMetadataKey]
	if !exists {
		return MeshsyncDeploymentModeUndefined
	}

	switch v := raw.(type) {
	case MeshsyncDeploymentMode:
		return v
	case string:
		return MeshsyncDeploymentModeFromString(v)
	default:
		return MeshsyncDeploymentModeUndefined
	}
}

// SetMeshsyncDeploymentModeToMetadata writes (or overwrites) the deployment
// mode entry on a connection's metadata map.
func SetMeshsyncDeploymentModeToMetadata(metadata core.Map, value MeshsyncDeploymentMode) {
	metadata[MeshsyncDeploymentModeMetadataKey] = value
}
