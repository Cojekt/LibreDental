package app

var isServerBuild = false
var isDevBuild = false

// IsServerBuild returns whether the application was compiled with -tags server.
func IsServerBuild() bool { return isServerBuild }

// IsDevBuild returns whether the application was compiled with -tags dev.
func IsDevBuild() bool { return isDevBuild }
