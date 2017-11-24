//+build !linux

package trace

func currentProcessTitle() (string, error) {
	// TODO(axw)
	return "", nil
}
