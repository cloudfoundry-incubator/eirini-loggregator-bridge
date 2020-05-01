ARG BASE_IMAGE=opensuse/leap

FROM golang:1.12 as build
ARG USER="SUSE CFCIBot"
ARG EMAIL=ci-ci-bot@suse.de
ARG DEBUG_TOOLS=false
ARG KUBECTL_VERSION=v1.18.2
ARG KUBECTL_ARCH=linux-amd64
ARG KUBECTL_CHECKSUM=ed36f49e19d8e0a98add7f10f981feda8e59d32a8cb41a3ac6abdfb2491b3b5b3b6e0b00087525aa8473ed07c0e8a171ad43f311ab041dcc40f72b36fa78af95
ARG GO111MODULE="on"

ENV GO111MODULE $GO111MODULE
WORKDIR /go/src/github.com/SUSE/eirini-loggregator-bridge

# Cache go modules if possible
ADD go.mod go.sum /go/src/github.com/SUSE/eirini-loggregator-bridge/
RUN if [ "${GO111MODULE}" = "on" ] ; then go mod download ; fi

ADD . /go/src/github.com/SUSE/eirini-loggregator-bridge/
RUN git config --global user.name ${USER}
RUN git config --global user.email ${EMAIL}
RUN bin/build
RUN if [ "$DEBUG_TOOLS" = "true" ] ; then \
    wget -O kubectl.tar.gz https://dl.k8s.io/$KUBECTL_VERSION/kubernetes-client-$KUBECTL_ARCH.tar.gz && \
    echo "$KUBECTL_CHECKSUM kubectl.tar.gz" | sha512sum --check --status && \
    tar xvf kubectl.tar.gz -C / && \
    cp -f /kubernetes/client/bin/kubectl /go/src/github.com/SUSE/eirini-loggregator-bridge/binaries/; fi

RUN mkdir -p tmp
RUN chmod a+rwx tmp

FROM $BASE_IMAGE
COPY --from=build /go/src/github.com/SUSE/eirini-loggregator-bridge/binaries/* /bin/
ENTRYPOINT ["/bin/eirini-loggregator-bridge"]
