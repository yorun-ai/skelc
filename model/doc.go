// Package model defines the parser-independent semantic data model for Skel
// contracts.
//
// Values in this package are produced by the root skelc package's Parse
// function and are the supported input for custom code generators. Names in
// SkelName fields are fully qualified, while Name fields contain the
// declaration's local name. Hash fields are deterministic compatibility hashes
// calculated by skelc.
package model
