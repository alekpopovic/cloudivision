FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/cloudivision-runner ./cmd/runner

FROM alpine/helm:4.0.4 AS helm

FROM golang:1.26-alpine
RUN apk add --no-cache ca-certificates chromium git nodejs npm \
  && adduser -D -H -u 65532 cloudivision \
  && mkdir -p /workspace /tmp/cloudivision \
  && chown -R 65532:65532 /workspace /tmp/cloudivision
COPY --from=build /out/cloudivision-runner /usr/local/bin/cloudivision-runner
COPY --from=helm /usr/bin/helm /usr/local/bin/helm
ENV CHROME_BIN=/usr/bin/chromium-browser
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/cloudivision-runner"]
