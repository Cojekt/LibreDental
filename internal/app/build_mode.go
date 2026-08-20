package app

var isServerBuild = false

// IsServerBuild returns whether the application was compiled with -tags server.
func IsServerBuild() bool { return isServerBuild }
