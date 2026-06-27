# Auth service

A secure, strictly-typed gRPC authentication service tailored for platform identity management.

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.26.4-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![gRPC](https://img.shields.io/badge/gRPC-API-244c5a?style=flat-square&logo=grpc)](https://grpc.io)
[![PostgreSQL](https://img.shields.io/badge/Database-PostgreSQL-336791?style=flat-square&logo=postgresql)](https://www.postgresql.org/)

## Overview

The auth service acts as the identity provider for the platform. It provides the mechanism to securely log into the admin console and manages the lifecycle of long-lived API keys used by external applications. The database schema is designed with multi-tenancy in mind but is currently locked to a single-tenant instance (ID=1) by design.



## Architecture

This section explains the technologies and physical layout of the service.

- **Framework**: Standard library `net` with `google.golang.org/grpc` for the RPC server
- **Database driver**: `github.com/jackc/pgx/v5/pgxpool` for PostgreSQL connection pooling
- **Validation**: Payload validation at the gRPC interceptor level using `buf.build/go/protovalidate`
- **Logging**: Idiomatic structured logging via `log/slog`

### Project structure

- `controller/`: gRPC server implementation and business logic
- `db/`: PostgreSQL connection management and pool setup
- `middlewares/`: gRPC interceptors
- `sql/`: Schema definitions, stored procedures, and role grants
- `utils/`: Helper functions for password hashing and timeouts
- `main.go`: Application entrypoint and dependency wiring
- `go.mod`: Dependency management

## Features

This section outlines the capabilities of the auth service.

- **Scalable identity**: Architecturally capable of multi-tenancy.
- **API key management**: Retrieve, validate, and securely rotate active prefixed API keys.
- **Payload validation**: Rejects malformed requests before they hit the controller.
- **Secure hash algorithms**: Uses `golang.org/x/crypto` for secure password hashing and comparison.

## Database security

The auth service leverages PostgreSQL with a strict focus on the principle of least privilege:

- Operates under the isolated `account` schema, managing the `creds` table
- Generates secure, prefixed API keys directly within the database layer using the `pgcrypto` extension
- The `auth_service` database role does **not** have direct `INSERT`, `UPDATE`, or `DELETE` permissions on tables
- All database mutations execute through securely defined stored functions

## Getting started

This section explains how to run the auth service locally.

### Prerequisites

- Go 1.26.4 or higher
- PostgreSQL 18+ database instance
- Protocol buffer compilation tools if modifying definitions

### Configuration

Export these variables directly in your shell environment:

| Variable | Description | Required |
| :--- | :--- | :---: |
| `POSTGRES_URL` | Connection string for the PostgreSQL database | **Yes** |
| `PORT` | The port on which the gRPC server will listen (Default: `7295`) | No |
| `INTERNAL_BIND_IP` | IP to bind the server to (Default: `0.0.0.0`) | No |

### Running locally

Run the service using the Go CLI:

```bash
go run main.go
```

To watch for file changes and automatically rebuild and restart the server, use [air](https://github.com/cosmtrek/air):

```bash
go install github.com/air-verse/air@latest
air
```

## Deployment

The service is containerized and leverages GitHub Actions (`.github/workflows/main.yml`) for automated builds and deployment to a remote VM using Podman.
