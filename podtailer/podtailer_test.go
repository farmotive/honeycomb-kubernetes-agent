package podtailer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/pkg/api/v1"
)

func TestDetermineLogPatternK8S110(t *testing.T) {
	t.Parallel()
	pod := &v1.Pod{}
	pod.Name = "foo"
	pod.UID = "123456"

	path, err := ioutil.TempDir("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(path)

	err = os.Mkdir(filepath.Join(path, "pods"), 0777)
	assert.Nil(t, err)
	err = os.Mkdir(filepath.Join(path, "pods", string(pod.UID)), 0777)
	assert.Nil(t, err)
	err = os.Mkdir(filepath.Join(path, "pods", string(pod.UID), "container1"), 0777)
	assert.Nil(t, err)
	err = os.Mkdir(filepath.Join(path, "pods", string(pod.UID), "container2"), 0777)
	assert.Nil(t, err)
	_, err = os.Create(filepath.Join(path, "pods", string(pod.UID), "container1", "0.log"))
	assert.Nil(t, err)
	_, err = os.Create(filepath.Join(path, "pods", string(pod.UID), "container2", "0.log"))
	assert.Nil(t, err)

	p, err := determineLogPattern(pod, path, false)
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(path, "pods", string(pod.UID), "*", "*"), p)
}

func TestDetermineLogPatternPreK8S110(t *testing.T) {
	t.Parallel()
	pod := &v1.Pod{}
	pod.Name = "foo"
	pod.UID = "123456"

	path, err := ioutil.TempDir("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(path)

	err = os.Mkdir(filepath.Join(path, "pods"), 0777)
	assert.Nil(t, err)
	err = os.Mkdir(filepath.Join(path, "pods", string(pod.UID)), 0777)
	assert.Nil(t, err)
	_, err = os.Create(filepath.Join(path, "pods", string(pod.UID), "container1_0.log"))
	assert.Nil(t, err)
	_, err = os.Create(filepath.Join(path, "pods", string(pod.UID), "container2_0.log"))
	assert.Nil(t, err)

	p, err := determineLogPattern(pod, path, false)
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(path, "pods", string(pod.UID), "*"), p)
}

func TestDetermineLogPatternLegacy(t *testing.T) {
	t.Parallel()
	pod := &v1.Pod{}
	pod.Name = "foo"
	pod.UID = "123456"
	pod.Namespace = "xyz"

	path, err := ioutil.TempDir("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(path)
	p, err := determineLogPattern(pod, path, true)
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(path, "containers", fmt.Sprintf("%s_%s_*.log", pod.Name, pod.Namespace)), p)
}

func TestDetermineFilterFuncLegacyLogs(t *testing.T) {
	t.Parallel()
	pod := &v1.Pod{}
	pod.Name = "foo"
	pod.Namespace = "default"
	expected := func(fileName string) bool {
		re := fmt.Sprintf(
			"^/var/log/containers/%s_%s_%s-.+\\.log",
			"foo",
			"default",
			"wibble",
		)
		ok, _ := regexp.Match(re, []byte(fileName))
		return ok
	}

	f := determineFilterFunc(pod, "wibble", true)
	// functions should have same output
	assert.Equal(
		t,
		expected("/var/log/containers/foo_default_wibble-abcdefg.log"),
		f("/var/log/containers/foo_default_wibble-abcdefg.log"),
	)

	assert.Equal(
		t,
		expected("/var/log/containers/foo_default_wobble-abcdefg.log"),
		f("/var/log/containers/foo_default_wobble-abcdefg.log"),
	)
}

func TestDetermineFilterFunc(t *testing.T) {
	t.Parallel()
	pod := &v1.Pod{}
	pod.UID = "123456"

	expected := func(fileName string) bool {
		re := fmt.Sprintf(
			"^/var/log/pods/%s/%s_[0-9]*\\.log",
			"123456",
			"wibble",
		)
		ok, _ := regexp.Match(re, []byte(fileName))
		return ok
	}

	f := determineFilterFunc(pod, "wibble", false)
	assert.Equal(
		t,
		expected("/var/log/pods/123456/wibble_0.log"),
		f("/var/log/pods/123456/wibble_0.log"),
	)

	assert.Equal(
		t,
		expected("/var/log/pods/123456/wobble_0.log"),
		f("/var/log/pods/123456/wobble_0.log"),
	)
}

func TestDetermineFilterFuncK8S1Dot10(t *testing.T) {
	t.Parallel()
	// Format changed in 1.10 to
	// /var/log/pods/<uid>/<containerName>/N.log
	// Handle it if we see it
	pod := &v1.Pod{}
	pod.UID = "123456"

	expected := func(fileName string) bool {
		re := fmt.Sprintf(
			"^/var/log/pods/%s/%s/[0-9]*\\.log",
			"123456",
			"wibble",
		)
		ok, _ := regexp.Match(re, []byte(fileName))
		return ok
	}

	f := determineFilterFunc(pod, "wibble", false)
	assert.Equal(
		t,
		expected("/var/log/pods/123456/wibble/0.log"),
		f("/var/log/pods/123456/wibble/0.log"),
	)

	assert.Equal(
		t,
		expected("/var/log/pods/123456/wobble/0.log"),
		f("/var/log/pods/123456/wobble/0.log"),
	)
}
