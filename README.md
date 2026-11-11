# Telegram Bot & Mini App Auto SDK

A high-performance, strongly-typed Go SDK and Auto Schema Builder for the **Telegram Bot API** and **Telegram Mini Apps (Web Apps)**.

---

## Features

- **Bot API Client (`pkg/telegram`)**: Complete type models and method implementations with HTTP connection pooling, timeout controls, JSON serialization, and multipart file upload support.
- **Mini App Suite (`pkg/miniapp`)**:
  - Full component builders: `BackButton`, `BottomButton`, `SettingsButton`, `HapticFeedback`, `CloudStorage`, `BiometricManager`, `LocationManager`, `DeviceStorage`, `SecureStorage`, and `Serverless`.
  - Cryptographic validation: HMAC-SHA256 signature verification and Ed25519 third-party public key verification.
  - Complete coverage of all 44 Mini App events and `ThemeParams`.
- **Auto Schema Builder (`pkg/schema`)**:
  - 79 registered schemas covering every Bot API method, Mini App component, type, and event.
  - Runtime payload validation against required fields, types, and constraints.
  - Standard JSON Schema (draft-07) export.
- **Zero External Dependencies**: Built entirely on Go standard library.

---

## Quick Start

### 1. Run Unit Tests

```bash
go test -v ./...
```

### 2. Run Live API Test Runner

Provide your bot token via the `-token` flag or the `TELEGRAM_BOT_TOKEN` environment variable (**do not hardcode tokens in source code**):

```bash
# Using flag:
go run cmd/testbot/main.go -token <YOUR_BOT_TOKEN>

# Using environment variable (PowerShell):
$env:TELEGRAM_BOT_TOKEN="<YOUR_BOT_TOKEN>"
go run cmd/testbot/main.go
```

---

## Documentation

- [**`docs/REPORT.md`**](docs/REPORT.md): Project notes, sources used, architecture, and live verification report.
- [**`docs/SDK_GUIDE.md`**](docs/SDK_GUIDE.md): Complete developer guide with Go code examples for Bot methods, Mini App components, InitData crypto validation, and Schema Builder usage.
