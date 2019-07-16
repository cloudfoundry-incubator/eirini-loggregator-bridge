FROM bitnami/kubectl as kubectl

FROM opensuse:leap
COPY --from=kubectl /opt/bitnami/kubectl/bin/kubectl /bin/kubectl
ADD binaries/eirini-loggregator-bridge /bin/eirini-loggregator-bridge
ENTRYPOINT ["/bin/eirini-loggregator-bridge"]

