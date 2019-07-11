package config_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEiriniConfigType(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ConfigType test Suite")
}
