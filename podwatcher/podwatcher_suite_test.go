package podwatcher_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEiriniPodWatcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PodWatcher test Suite")
}
