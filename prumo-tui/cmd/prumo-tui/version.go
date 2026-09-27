package main

// Version is set at build time with -ldflags. An unset value is honest about
// being a development build rather than pretending to be a release.
var Version = "dev"
