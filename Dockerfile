FROM cgr.dev/chainguard/go:latest AS builder
ENV CGO_ENABLED=0
WORKDIR /workspace
COPY go.mod .
COPY go.sum .
COPY . .
RUN go mod download && go build .

FROM ghcr.io/anchore/syft:latest AS sbomgen
COPY --from=builder /workspace/whisper-to-graphite /usr/bin/whisper-to-graphite
RUN ["/syft", "--output", "spdx-json=/tmp/whisper-to-graphite.spdx.json", "/usr/bin/whisper-to-graphite"]

FROM cgr.dev/chainguard/static:latest
WORKDIR /tmp
COPY --from=builder /workspace/whisper-to-graphite /usr/bin/
COPY --from=sbomgen /tmp/whisper-to-graphite.spdx.json /var/lib/db/sbom/whisper-to-graphite.spdx.json
ENTRYPOINT ["/usr/bin/whisper-to-graphite"]
LABEL org.opencontainers.image.title="whisper-to-graphite"
LABEL org.opencontainers.image.description="Read and send metrics from whisper files to graphite"
LABEL org.opencontainers.image.url="https://github.com/tgragnato/whisper-to-graphite/"
LABEL org.opencontainers.image.source="https://github.com/tgragnato/whisper-to-graphite/"
LABEL license="MIT"
LABEL io.containers.autoupdate=registry
