module github.com/SUSE/eirini-loggregator-bridge

require (
	code.cloudfoundry.org/go-diodes v0.0.0-20180905200951-72629b5276e3 // indirect
	code.cloudfoundry.org/go-loggregator v7.4.0+incompatible
	code.cloudfoundry.org/rfc5424 v0.0.0-20180905210152-236a6d29298a // indirect
	github.com/SUSE/eirinix v0.2.1-0.20200719200104-81b184602339
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/spf13/cobra v0.0.7
	github.com/spf13/viper v1.6.3
	k8s.io/api v0.0.0-20200404061942-2a93acf49b83
	k8s.io/apimachinery v0.0.0-20200410010401-7378bafd8ae2
	k8s.io/client-go v0.0.0-20200330143601-07e69aceacd6
)

replace code.cloudfoundry.org/cf-operator => code.cloudfoundry.org/quarks-operator v1.0.1-0.20200413083459-fb39a29ad746

go 1.13
