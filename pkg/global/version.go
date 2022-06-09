package global

import "os"

var (
	// NucleusBinaryVersion Nucleus version
	NucleusBinaryVersion = os.Getenv("VERSION")
	// SynapseBinaryVersion Synapse version
	SynapseBinaryVersion = os.Getenv("VERSION")
)
