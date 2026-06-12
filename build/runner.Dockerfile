FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/cloudivision-runner ./cmd/runner

FROM alpine:3.22
RUN apk add --no-cache ca-certificates git \
  && adduser -D -H -u 65532 cloudivision \
  && mkdir -p /workspace \
  && chown 65532:65532 /workspace
COPY --from=build /out/cloudivision-runner /usr/local/bin/cloudivision-runner
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/cloudivision-runner"]
