# Auth Service

A reliable and scalable gRPC authentication service written in Go.

[![Go Version](https://img.shields.io/badge/Go->=1.25-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![gRPC](https://img.shields.io/badge/gRPC-API-244c5a?style=flat-square&logo=grpc)](https://grpc.io)
[![PostgreSQL](https://img.shields.io/badge/Database-PostgreSQL-336791?style=flat-square&logo=postgresql)](https://www.postgresql.org/)

## Overview

The Auth Service is a core infrastructure microservice responsible for handling user authentication and API key lifecycle management. It acts as the gatekeeper for identity within the platform, exposing a strictly-typed gRPC API that other microservices can consume. Built with Go, it emphasizes high performance, secure defaults, and robust error handling.

This service utilizes PostgreSQL as its source of truth, managing passwords securely with hashing and exposing API keys that can be rotated dynamically via stored procedures.

## Architecture & Tech Stack

- **Framework**: Standard library `net` with `google.golang.org/grpc` for the RPC server.
- **Database Driver**: Uses `github.com/jackc/pgx/v5/pgxpool` for high-performance PostgreSQL connection pooling.
- **Validation**: Payload validation happens implicitly at the gRPC interceptor level using `buf.build/go/protovalidate`.
- **Logging**: Idiomatic structured logging is implemented via `log/slog`.

### Project Structure

```text
.
├── controller/    # Contains the gRPC server implementation and business logic (handlers.go)
├── db/            # PostgreSQL connection management and pool setup (postgres.go, setup.go)
├── middlewares/   # gRPC interceptors, including payload validation
├── sql/           # Schema definitions, stored procedures, and role grants
├── utils/         # Helper functions for password hashing, comparison, and timeouts
├── main.go        # Application entrypoint and dependency wiring
└── go.mod         # Dependency management
```

## Features

- 🔐 **Authentication & Registration**: Secure login flow. Automatically registers a user if the database is empty.
- 🔑 **API Key Management**: Retrieve, validate, and securely rotate API keys.
- 🛡️ **Payload Validation**: Automatically rejects malformed requests before they hit the controller using `protovalidate` interceptors.
- ⚡ **Connection Pooling**: Reuses database connections effectively via `pgxpool`.
- 🚦 **Graceful Shutdown**: Intercepts `SIGTERM` and `SIGINT`, allows ongoing requests to finish, and safely closes database connections.
- 🔒 **Secure Hash Algorithms**: Uses `golang.org/x/crypto` for secure password comparison and hashing.

## API Summary

The service implements a gRPC interface focused strictly on identity and authentication:
- **Login / Registration**: Initializing the admin identity or authenticating existing credentials.
- **Key Validation**: Confirming the active state of securely generated, prefixed API keys.
- **Key Rotation**: Securely invalidating and rolling over API credentials.

## Database Architecture

- **Dedicated Schema**: Operates under the `account` schema, managing the `creds` table.
- **Secure Key Generation**: Uses the `pgcrypto` extension to generate secure, prefixed API keys directly within the database layer.
- **Restricted Access**: Connects using the `auth_service` database role, which does **not** have direct `INSERT`, `UPDATE`, or `DELETE` permissions on tables.
- **Stored Procedures**: All writes to the database are strictly executed through securely defined stored functions (e.g., `account.create`).

## Getting Started

### Prerequisites

- [Go](https://go.dev/dl/) 1.25 or higher
- A running [PostgreSQL](https://www.postgresql.org/download/) 18+ database instance
- Protocol buffer compilation tools (e.g., `buf`) if modifying definitions.

### Configuration

The service relies on environment variables for configuration. You should export these directly in your shell environment.

| Variable | Description | Default | Required |
| :--- | :--- | :--- | :---: |
| `POSTGRES_URL` | Connection string for the PostgreSQL database | - | **Yes** |
| `PORT` | The port on which the gRPC server will listen | `7295` | No |

### Polyrepo Local Setup

This project uses a polyrepo architecture. Services like `auth`, `manager`, and `gateway` rely on the `common` repository. To run this locally without dependency errors, clone all repositories side-by-side into the same parent directory so that relative paths resolve correctly.

Example local setup:
```text
infrastructure/
├── common/
├── auth/
├── manager/
└── gateway/
```

### Running the Service

For standard execution, use the Go CLI:

```bash
go run main.go
```

#### Hot Reloading for Local Development

To watch for file changes and automatically rebuild and restart the server, you can use [air](https://github.com/cosmtrek/air). Since `air` is written purely in Go, it compiles to a single binary and runs natively on Windows, macOS, and Linux without any OS-specific shell scripts.

```bash
go install github.com/air-verse/air@latest
air
```

## CI/CD & Production Deployment

The service is fully containerized and leverages GitHub Actions (`.github/workflows/main.yml`) for automated builds and deployment.

### 1. Server Initialization
Before the CI/CD pipeline can deploy the application for the first time, you must initialize the host environment on your VM.

Create the shared network that the containers will use to communicate:
```bash
podman network create infra-network
```

Create the environment variable configuration file:
```bash
touch ~/auth-service-production.conf
# Edit the file to include your POSTGRES_URL
```

### 2. Automated Build (GHCR)
The workflow uses Docker Buildx with QEMU to cross-compile a `linux/arm64` container image. 
Because the project depends on private GitHub modules (`common`), the workflow injects a `PAT_TOKEN` secret during the build stage to securely download the private contracts. The resulting image is pushed to the GitHub Container Registry (`ghcr.io/aditya-0011/auth-service`).

### 3. Production Execution (Podman)
In production, the service is deployed to a remote Linux VM using **Podman**. The CI pipeline automatically SSHs into the server, pulls the latest GHCR image, and seamlessly swaps the running container.

The container is orchestrated with the following constraints:
- Attached to the shared internal network (`--network infra-network`).
- Reads configuration purely from the host-level environment file (`--env-file ~/auth-service-production.conf`).
- Configured to restart automatically (`--restart unless-stopped`).
