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
- `internal/password/`: argon2id hashing (PHC format, upgrades legacy hashes on login)
- `internal/secret/`: API key and recovery code generation and hashing
- `sql/`: Schema definitions, stored procedures, and role grants
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
| `INTERNAL_BIND_IP` | Private IP to listen on (the address the gateway dials). No default | **Yes** |
| `INTERNAL_AUTH_TOKEN` | Shared secret (32+ chars) the gateway must send as `authorization: Bearer <token>`. Same value on every service and the gateway | **Yes** |
| `INTERNAL_TLS_CERT` / `INTERNAL_TLS_KEY` | PEM certificate and key; enables TLS on the gRPC listener when both are set | No |
| `RPC_TIMEOUT_MS` | Upper bound per RPC (default `5000`). A shorter deadline sent by the gateway wins | No |
| `AUTH_ALLOW_BOOTSTRAP` | `true` lets a login with an unknown email create the administrator when no administrator exists. Leave unset in normal operation | No |

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

## Users, roles and projects

- **Roles.** One administrator (role 1) and any number of tenants (role 2). The administrator can open every project and manage users and projects from the Console. Tenants can open only the projects granted to them that are active.
- **Projects.** `account.project` holds every app shown in the Console (slug, name, URL, icon, status). Status is active, maintenance (visible, but only the administrator can use it) or hidden. The gateway guards the `manager` and `wallet` APIs by project slug.
- **Creating users.** The administrator adds a user by email and picks their projects. The service returns a temporary password, shown once and valid for 7 days. At first login the user must choose a new password and receives 10 backup codes.
- **Administrator actions.** Reset a user's password (issues a new temporary password and signs them out everywhere), disable or enable them, change their projects, or delete them. Data the user created in other services is not deleted. The administrator account itself cannot be disabled, deleted or re-scoped.
- **API key.** One per user, available to the administrator and to users with active Manager access (Console > API Key). A portfolio uses it to read that user's Manager content and post contact messages. Stored as a SHA-256 hash and shown once when created or rotated; it can be revoked. `key_origins` lists up to 10 portfolio origins allowed to use the key from a browser. Losing Manager access (revoked, disabled, or Manager in maintenance) disables the key at once. Migration `003_api_key_origins.sql` adds the column.

## Account recovery

There is no email or SMS reset:

- **Backup codes.** Ten one-time codes are issued at first login and can be regenerated or destroyed from the Console's Security page (both need the current password). "Forgot password?" on the login page takes an email, one code and a new password. A reset signs the account out everywhere.
- **Tenants without codes.** The administrator resets their password from Console > Users.
- **Administrator without codes (last resort, needs server access).** Delete the administrator row from `account.creds`, set `AUTH_ALLOW_BOOTSTRAP=true`, restart, log in (this recreates the administrator with that email and password), then unset the variable and restart again. Tenants keep working meanwhile. Only an email with no account can claim the empty administrator role, and only while the variable is set.

## Database migrations

`sql/script.sql` is the idempotent fresh-install schema. Existing databases run the numbered files in `sql/migrations/` first, in order, then `script.sql`.

## Deployment

The service is containerized and leverages GitHub Actions (`.github/workflows/main.yml`) for automated builds and deployment to a remote VM using Podman.
