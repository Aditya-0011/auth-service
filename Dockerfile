ARG GO_VERSION=1.25.3
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
ENV GOPRIVATE="github.com/Aditya-0011/*"

COPY go.mod go.sum ./
RUN --mount=type=secret,id=GITHUB_TOKEN \
    git config --global url."https://$(cat /run/secrets/GITHUB_TOKEN):x-oauth-basic@github.com/Aditya-0011".insteadOf "https://github.com/Aditya-0011" && \
    go mod download && \
    git config --global --remove-section url."https://$(cat /run/secrets/GITHUB_TOKEN):x-oauth-basic@github.com/Aditya-0011"

COPY . .

ARG TARGETOS TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags "-s -w" -o bin-auth main.go

FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/bin-auth ./auth

EXPOSE 7295

ENTRYPOINT ["./auth"]
