package fsutil

import (
	"k8s.io/klog"
	"os"
	"path"
)

func AssertExists(filename string) string {
	ok, err := Exists(filename)
	if !ok || err != nil {
		klog.Errorf("File is not reachable: %s", filename)
	}
	return filename
}

func EnsureExists(dirs ...string) string {
	target := path.Join(dirs...)
	if err := os.MkdirAll(target, 0755); err != nil {
		klog.Errorf("Directory is not reachable:%s", target)
	}
	return target
}

func Exists(name string) (bool, error) {
	_, err := os.Stat(name)
	if err != nil && os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}
