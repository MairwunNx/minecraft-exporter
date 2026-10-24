# Builds on the build machine's own architecture and cross-compiles, so multi-arch images need no emulation.
FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine AS build
ARG TARGETOS TARGETARCH VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o /minecraft-exporter .

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /minecraft-exporter /minecraft-exporter
EXPOSE 10059
USER nonroot
ENTRYPOINT ["/minecraft-exporter"]
