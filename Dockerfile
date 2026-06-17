ARG GO_VERSION=1.26.4
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

RUN apk add --no-cache git tzdata ca-certificates

WORKDIR /app
ENV GOPRIVATE="github.com/Aditya-0011/*"

COPY go.mod go.sum ./
RUN --mount=type=secret,id=GITHUB_TOKEN \
    git config --global url."https://$(cat /run/secrets/GITHUB_TOKEN):x-oauth-basic@github.com/Aditya-0011".insteadOf "https://github.com/Aditya-0011" && \
    go mod download && \
    git config --global --remove-section url."https://$(cat /run/secrets/GITHUB_TOKEN):x-oauth-basic@github.com/Aditya-0011"

COPY . .

ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags "-s -w" -trimpath -tags netgo -o bin-auth main.go

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/bin-auth /auth

EXPOSE 7295

ENV GOMAXPROCS=1 \
    GOMEMLIMIT=850MiB \
    GOGC=200

ENTRYPOINT ["/auth"]
