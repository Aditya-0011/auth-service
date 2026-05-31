<div align="center">

# Auth Service
*A reliable and scalable gRPC authentication service written in Go*

[![Go Version](https://img.shields.io/badge/Go->=1.25-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![gRPC](https://img.shields.io/badge/gRPC-API-244c5a?style=flat-square&logo=grpc)](https://grpc.io)
[![PostgreSQL](https://img.shields.io/badge/Database-PostgreSQL-336791?style=flat-square&logo=postgresql)](https://www.postgresql.org/)

⭐ If you find this service useful, please star the repository!

[Overview](#overview) • [Architecture & Tech Stack](#architecture--tech-stack) • [Features](#features) • [Getting Started](#getting-started) • [Polyrepo Local Setup](#polyrepo-local-setup) • [API Summary](#api-summary) • [Database Architecture](#database-architecture)

</div>

## Overview

The **Auth Service** is a core infrastructure microservice responsible for handling user authentication and API key lifecycle management. It acts as the gatekeeper for identity within the platform, exposing a strictly-typed gRPC API that other microservices can consume. Built with Go, it emphasizes high performance, secure defaults, and robust error handling.

This service utilizes PostgreSQL as its source of truth, managing passwords securely with hashing and exposing API keys that can be rotated dynamically via stored procedures.

## Architecture & Tech Stack

- **Framework**: Standard library `net` with `google.golang.org/grpc` for the RPC server.
- **Database Driver**: Uses `github.com/jackc/pgx/v5/pgxpool` for high-performance PostgreSQL connection pooling.
- **Validation**: Payload validation happens implicitly at the gRPC interceptor level using `buf.build/go/protovalidate`.
- **Logging**: Idiomatic structured logging is implemented via `log/slog`.
- **Contracts**: Shares proto definitions through a common Go module (`github.com/Aditya-0011/common/contracts/go/auth`).

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

- 🔐 **Authentication & Registration**: Secure login flow. Automatically registers a user if the database is empty (currently limited to a single administrative user setup).
- 🔑 **API Key Management**: Retrieve, validate, and securely rotate API keys.
- 🛡️ **Payload Validation**: Automatically rejects malformed requests before they hit the controller using `protovalidate` interceptors.
- ⚡ **Connection Pooling**: Reuses database connections effectively via `pgxpool`.
- 🚦 **Graceful Shutdown**: Intercepts `SIGTERM` and `SIGINT`, allows ongoing requests to finish, and safely closes database connections with a hard timeout fallback of 5 seconds.
- 🔒 **Secure Hash Algorithms**: Uses `golang.org/x/crypto` for secure password comparison and hashing.

## Getting Started

### Prerequisites

- [Go](https://go.dev/dl/) 1.25 or higher
- A running [PostgreSQL](https://www.postgresql.org/download/) 18+ database instance
- The `common` contracts module (see Polyrepo setup below).

### Installation

Clone the repository:
```bash
git clone https://github.com/Aditya-0011/auth-service.git auth
cd auth
go mod tidy
```

### Configuration

The service relies on environment variables for configuration. You can export these directly in your shell or place them in a `.env` file at the root of the project.

| Variable | Description | Default | Required |
| :--- | :--- | :--- | :---: |
| `POSTGRES_URL` | Connection string for the PostgreSQL database (e.g. `postgres://user:pass@host/db`) | - | **Yes** |
| `PORT` | The port on which the gRPC server will listen | `7295` | No |

### Running the Service

For standard execution, use the Go CLI:

```bash
go run main.go
```

> [!TIP]
> If you are working in a local development environment, you might have the `run.ps1` script configured with [Air](https://github.com/cosmtrek/air) for live-reloading. You can execute `./run.ps1` to automatically rebuild the server on code changes.

## Polyrepo Local Setup

This project uses a polyrepo architecture. The `auth` service is in its own repository, but it depends on the schemas defined in the `common` repository.

> [!IMPORTANT]
> When developing locally, keep all repositories (`common`, `auth`, `manager`) in the same parent folder. Use a Go Workspace file (`go.work`) in the parent directory to safely resolve the local `common` module without altering `go.mod` files.

Example local setup:
```text
infrastructure/
├── common/
├── auth/
├── manager/
└── go.work # Should include 'use ./common', 'use ./auth', 'use ./manager'
```

## API Summary

The service implements a gRPC interface focused strictly on identity and authentication:
- **Login / Registration**: Initializing the admin identity or authenticating existing credentials.
- **Key Validation**: Confirming the active state of UUID-based API keys.
- **Key Rotation**: Securely invalidating and rolling over API credentials.

## Database Architecture

The auth service leverages PostgreSQL with a strong focus on the principle of least privilege.
- **Dedicated Schema**: It operates under the `account` schema, managing the `creds` table.
- **Restricted Access**: The backend application connects using the `auth_service` database role, which does **not** have direct `INSERT`, `UPDATE`, or `DELETE` permissions on tables.
- **Stored Procedures**: All writes to the database (such as user creation or key rotation) are strictly executed through securely defined stored functions (`account.create`, `account.rotate_key`).
