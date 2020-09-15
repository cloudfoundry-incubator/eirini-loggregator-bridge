package config_test

import (
	configpkg "code.cloudfoundry.org/eirini-loggregator-bridge/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("Validate", func() {
		var config configpkg.ConfigType

		validConfig := configpkg.ConfigType{
			Namespace:           "some_namespace",
			LoggregatorEndpoint: "some_endpoint",
			LoggregatorCAPath:   "ca_path",
			LoggregatorCertPath: "cert_path",
			LoggregatorKeyPath:  "key_path",
		}

		Context("when namespace is missing", func() {
			BeforeEach(func() {
				config = validConfig
				config.Namespace = ""
			})
			It("returns an error", func() {
				err := config.Validate()
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(Equal("namespace is missing from configuration"))
			})
		})

		Context("when loggregator-endpoint is missing", func() {
			BeforeEach(func() {
				config = validConfig
				config.LoggregatorEndpoint = ""
			})
			It("returns an error", func() {
				err := config.Validate()
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(Equal("loggregator-endpoint is missing from configuration"))
			})
		})
		Context("when loggregator-ca-path is missing", func() {
			BeforeEach(func() {
				config = validConfig
				config.LoggregatorCAPath = ""
			})
			It("returns an error", func() {
				err := config.Validate()
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(Equal("loggregator-ca-path is missing from configuration"))
			})
		})
		Context("when loggregator-cert-path is missing", func() {
			BeforeEach(func() {
				config = validConfig
				config.LoggregatorCertPath = ""
			})
			It("returns an error", func() {
				err := config.Validate()
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(Equal("loggregator-cert-path is missing from configuration"))
			})
		})
		Context("when loggregator-key-path is missing", func() {
			BeforeEach(func() {
				config = validConfig
				config.LoggregatorKeyPath = ""
			})
			It("returns an error", func() {
				err := config.Validate()
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(Equal("loggregator-key-path is missing from configuration"))
			})
		})
	})
})
