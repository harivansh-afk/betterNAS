//go:build !(android || darwin || dragonfly || freebsd || illumos || ios || linux || netbsd || openbsd || solaris)

package nodeagent

func detectCapacityBytes(string) *int64 {
	return nil
}
