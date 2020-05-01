# Eirini Loggregator Bridge

This project is a tool that reads logs from pods in a Kubernetes cluster namespace and streams them to a specified Loggregator endpoint. The main purpose of this project is to be used as a logging solution for [Eirini](https://github.com/cloudfoundry-incubator/eirini)

## Build

```make```

will vet, lint, test and build the project. There are separate Make targets for
these tasks so check in the Makefile for details.

## Usage

You need to create a config yaml file to be able to use this tool. The needed options are:

- loggregator-ca-path
  This is the path to the CA that signs the Loggregator certificate
- loggregator-cert-path
  This is the SSL certificate to talk to the loggregator over TLS
- loggregator-key-path
  This is the private key for the TLS communication
- loggregator-endpoint
  This is the endpoint to your Loggregator instance. E.g.
  "doppler-doppler.scf.svc.cluster.local:8082"
- namespace
  This is the namespace where Eirini deploys applications

Example config.yaml:

```
loggregator-ca-path: /certs/ca
loggregator-cert-path: /certs/cert
loggregator-key-path: /certs/key
loggregator-endpoint: doppler-doppler.scf.svc.cluster.local:8082
namespace: eirini
```

Then run this tool:

```
./eirini-loggregator-bridge --config config.yaml
```

if you are running it from outside the Kubernetes cluster you can specify the path
to your kubeconfig like:

```
./eirini-loggregator-bridge --config config.yaml --kubeconfig ~/.kube/config
```
In that case though you have to make sure your loggregator endpoint is accessible
from outside the cluster.


## Development

### Building docker image with local dependencies

The docker image can be built with:

```
make build-image
```

To use local dependencies (e.g. to use a locally modified `eirinix` for testing),
update the go module dependencies as normal:

```
go mod edit -replace github.com/SUSE/eirinix=../eirinix
```

Then, to build the docker image:

```
VENDOR=on make build-image
```

### Debugging

You can increase the log level of the tool by setting the `EIRINI_LOGGREGATOR_BRIDGE_LOGLEVEL`
environment variable. Allowed values are: "DEBUG", "INFO", "WARN", "ERROR" (Default is "WARN")
