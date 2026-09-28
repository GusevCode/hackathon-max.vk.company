FROM golang:1.25-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates git

COPY src/backend/go.mod src/backend/go.sum ./
RUN go mod download

COPY src/backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/control-bot ./cmd/control-bot

FROM alpine:3.22

RUN apk add --no-cache ca-certificates wget \
  && addgroup -S app \
  && adduser -S -G app app

COPY --from=build /out/control-bot /usr/local/bin/control-bot
COPY infra/certs/russian_trusted_root_ca_pem.crt /usr/local/share/ca-certificates/russian_trusted_root_ca.crt
COPY infra/certs/russian_trusted_sub_ca_pem.crt /usr/local/share/ca-certificates/russian_trusted_sub_ca.crt

RUN update-ca-certificates

EXPOSE 8080
USER app
ENTRYPOINT ["/usr/local/bin/control-bot"]
