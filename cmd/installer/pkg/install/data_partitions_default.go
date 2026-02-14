//go:build !chubo

package install

func forceDataPartitions() bool {
	return false
}
