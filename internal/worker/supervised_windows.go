//go:build windows

package worker

// Windows Job Object kill-on-close already provides a kernel parent-lifetime
// guard; creating an additional helper would add no ownership protection.
func startSupervised(spec launchSpec) (process, error) {
	if spec.inspection != nil {
		return startWindowsInspection(spec)
	}
	return startProcess(spec)
}
