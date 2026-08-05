// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package brand carries the project identity every tool and generated artifact
// keys off.
//
// Four identifiers derive from one token, and all four have to agree: the
// directive namespace a consumer writes in their source, the header markers
// eidos renders into generated files, the project-local state directory, and
// the config filename. Any two of them disagreeing is a silent failure —
// directives that parse but are never read, a config file nothing discovers,
// artifacts written where nothing looks for them.
//
// They are constants rather than configuration on purpose. eidos takes the
// brand once at program start and never from a flag or a config key, because
// changing it mid-project would orphan every artifact already on disk.
package brand

// Name is the project identity. Everything else here derives from it, so a
// rename is one edit rather than a search.
const Name = "testkit"

// DirectivePrefix is the namespace consumers write directives under:
//
//	//testkit:mixin idempotent
//
// eidos's parser takes the prefix as configuration, so this is what testkit
// passes it. The annotator reads nothing written under any other namespace,
// which is why a corpus and a CLI disagreeing here produces a corpus that
// stamps nothing (docs/adr/0016).
const DirectivePrefix = Name

// ConfigFile is the project config filename, discovered by walking up from the
// working directory.
//
// One name, no alternates: a second accepted spelling is permanent ambiguity
// in every lookup, error message, and document (docs/adr/0009). The `.yaml`
// extension is the one the YAML specification recommends.
const ConfigFile = "." + Name + ".yaml"

// StateDir is the project-local directory holding the manifest and cached
// pipeline state. It is derived rather than written out so it cannot drift
// from [ConfigFile] under a rename.
const StateDir = "." + Name
