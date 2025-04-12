package vulkan_errors

import "errors"

var ErrGitNotInstalled = errors.New("git not installed")
var ErrGitCloneFailed = errors.New("git clone failed")
