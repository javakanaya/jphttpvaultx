# jphttpvaultx

A lightweight, zero-dependency Go HTTP client for the [HashiCorp Vault Lambda Extension](https://developer.hashicorp.com/vault/docs/platform/aws/aws-lambda) proxy.

The extension runs as a sidecar inside the AWS Lambda execution environment and exposes a local proxy at `http://127.0.0.1:8200`. It handles Vault authentication (via AWS IAM) transparently — **no tokens or AppRoles needed in your code**.

---

## Installation

```bash
go get github.com/javakanaya/jphttpvaultx
```

---

## Quick Start

```go
import jphttpvaultx "github.com/javakanaya/jphttpvaultx"

// Initialise once at cold-start (outside the handler).
var vault = jphttpvaultx.New(
    jphttpvaultx.WithKVMount("static-secret"), // your KV v2 mount name
)
```

---

## Configuration Options

| Option                | Default                 | Description                                                 |
| --------------------- | ----------------------- | ----------------------------------------------------------- |
| `WithProxyAddr(addr)` | `http://127.0.0.1:8200` | Override the extension proxy address                        |
| `WithKVMount(mount)`  | `secret`                | KV v2 secrets engine mount path                             |
| `WithNamespace(ns)`   | _(none)_                | Vault Enterprise namespace (`X-Vault-Namespace`)            |
| `WithTimeout(d)`      | `5s`                    | HTTP client timeout                                         |
| `WithHTTPClient(hc)`  | _(built-in)_            | Bring your own `*http.Client` (custom TLS, transport, etc.) |

---

## Usage

### Initialise in Lambda (cold-start)

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/aws/aws-lambda-go/lambda"
    jphttpvaultx "github.com/javakanaya/jphttpvaultx"
)

var vault *jphttpvaultx.Client

func init() {
    vault = jphttpvaultx.New(
        jphttpvaultx.WithKVMount(envOr("VAULT_KV_MOUNT", "static-secret")),
        jphttpvaultx.WithNamespace(os.Getenv("VAULT_NAMESPACE")), // omit for Vault OSS
        jphttpvaultx.WithTimeout(3*time.Second),
    )
}

func envOr(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}

func handler(ctx context.Context) error {
    // fetch a single secret key
    apiKey, err := vault.GetSecret(ctx, "my-service/config", "api_key")
    if err != nil {
        return err
    }
    log.Println("got api_key:", apiKey)

    // fetch database credentials
    creds, err := vault.GetDatabaseCredentials(ctx, "payments-db")
    if err != nil {
        return err
    }
    log.Printf("db user: %s", creds.Username)

    return nil
}

func main() { lambda.Start(handler) }
```

---

### `GetSecret` — fetch a single key

```go
// Reads <mount>/data/my-service/config and returns the value of "api_key".
val, err := vault.GetSecret(ctx, "my-service/config", "api_key")
```

---

### `GetSecretMap` — fetch all keys

```go
// Returns the full inner data map of the secret.
m, err := vault.GetSecretMap(ctx, "my-service/config")
region := m["region"].(string)
```

---

### `GetSecretKey` — fetch a dedicated `secret_key` field

Reads the `secret_key` field from `<mount>/data/<path>`. Accepts both flat and `secrets`-wrapped layouts (see [Secret Layouts](#secret-layouts) below).

```go
signingKey, err := vault.GetSecretKey(ctx, "jwt/signing")
```

---

### `GetDatabaseCredentials` — typed DB credentials

Reads `<mount>/data/<path>` and returns a `*DatabaseCredentials` with `Username` and `Password`. Accepts both flat and `secrets`-wrapped layouts.

```go
creds, err := vault.GetDatabaseCredentials(ctx, "payments-db")
if err != nil {
    log.Fatal(err)
}

db, err := sql.Open("postgres", fmt.Sprintf(
    "postgres://%s:%s@host:5432/mydb",
    creds.Username, creds.Password,
))
```

---

### `GetThirdPartyAppCredential` — typed third-party credentials

Reads `<mount>/data/<path>` and returns a `*ThirdPartyAppCredentials` with `Email` and `Password`. Accepts both flat and `secrets`-wrapped layouts.

```go
creds, err := vault.GetThirdPartyAppCredential(ctx, "stripe")
if err != nil {
    log.Fatal(err)
}
log.Printf("stripe user: %s", creds.Email)
```

---

## Secret Layouts

All typed helpers (`GetDatabaseCredentials`, `GetThirdPartyAppCredential`, `GetSecretKey`) accept **both** flat and `secrets`-wrapped Vault secret layouts:

**Flat** — fields stored directly at the top level:

```json
{ "username": "admin", "password": "s3cr3t" }
```

**Wrapped** — fields nested under a `secrets` key:

```json
{ "secrets": { "username": "admin", "password": "s3cr3t" } }
```

If the `secrets` key is present and contains a map, the inner map is used. Otherwise the top-level data is used as-is.

---

## Vault KV v2 Envelope

This client targets **KV v2** secrets. The Vault API response envelope looks like:

```
GET /v1/<mount>/data/<path>
{
  "data": {
    "data": {
      "username": "admin",
      "password": "s3cr3t"
    },
    "metadata": { ... }
  }
}
```

`readKV` (used internally by all helpers) unwraps the outer envelope so you always work with the **inner `data` map** directly.

---

## Environment Variables (recommended)

| Variable           | Description                                               |
| ------------------ | --------------------------------------------------------- |
| `VAULT_KV_MOUNT`   | KV v2 mount name (e.g. `static-secret`)                   |
| `VAULT_NAMESPACE`  | Vault Enterprise namespace (omit for OSS)                 |
| `VAULT_PROXY_ADDR` | Override proxy address (default: `http://127.0.0.1:8200`) |
