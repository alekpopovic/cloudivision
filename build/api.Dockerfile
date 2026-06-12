FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/cloudivision-api ./cmd/api

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -H -u 65532 cloudivision
COPY --from=build /out/cloudivision-api /usr/local/bin/cloudivision-api
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/cloudivision-api"]
