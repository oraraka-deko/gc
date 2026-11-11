# Telegram Bot & Mini App Auto SDK

A high-performance, strongly-typed Go SDK, Auto Schema Builder, and Updates Processing Core for the **Telegram Bot API** and **Telegram Mini Apps (Web Apps)**.

---

## Features

- **Updates Processing Core (`pkg/telegram`)**:
  - Full support for all 30 update types from canonical Telegram specifications.
  - Unified [`Dispatcher`](pkg/telegram/dispatcher.go) with middleware, command parsing, and typed routing.
  - Production-ready [`LongPoller`](pkg/telegram/poller.go) with automatic offset tracking, zero duplicates, and worker pools.
  - Standard [`WebhookHandler`](pkg/telegram/webhook.go) with constant-time `X-Telegram-Bot-Api-Secret-Token` validation.
- **Bot API Client (`pkg/telegram`)**: Complete type models and method implementations with HTTP connection pooling, timeout controls, JSON serialization, and multipart file upload support.
- **Mini App Suite (`pkg/miniapp`)**:
  - Full component builders: `BackButton`, `BottomButton`, `SettingsButton`, `HapticFeedback`, `CloudStorage`, `BiometricManager`, `LocationManager`, `DeviceStorage`, `SecureStorage`, and `Serverless`.
  - Cryptographic validation: HMAC-SHA256 signature verification and Ed25519 third-party public key verification.
  - Complete coverage of all 44 Mini App events and `ThemeParams`.
- **Auto Schema Builder (`pkg/schema`)**:
  - 79 registered schemas covering every Bot API method, Mini App component, type, and event.
  - Runtime payload validation against required fields, types, and constraints.
  - Standard JSON Schema (draft-07) export.
- **Production Example Bot (`examples/crypto_converter_bot`)**:
  - Realtime rates and conversions for **TON/Gram**, **TRX/Tron**, **USDT**, **BTC**, and **ETH**.
  - Natural language parsing for `1 ton`, `1grAm`, `500ton tron`, `tron 500`, `usdt tron`, etc.
  - Multi-source fallback: TonAPI ➔ Binance ➔ CoinGecko ➔ OKX.
  - Works across **Direct Chat**, **Groups**, and **Inline Mode**.
- **Zero External Dependencies**: Built entirely on the Go standard library.

---

## Quick Start

### 1. Run Unit Tests

```bash
go test -v ./...
```

### 2. Run Live Crypto & TON Converter Bot Example

Provide your bot token via the `-token` flag or the `TELEGRAM_BOT_TOKEN` environment variable (**do not hardcode tokens in source code**):

```bash
# Using flag:
go run ./examples/crypto_converter_bot -token <YOUR_BOT_TOKEN>

# Using environment variable (PowerShell):
$env:TELEGRAM_BOT_TOKEN="<YOUR_BOT_TOKEN>"
go run ./examples/crypto_converter_bot
```

### 3. Run Live SDK Verification Tool

```bash
go run cmd/testbot/main.go -token <YOUR_BOT_TOKEN>
```

---

## Documentation

- [**`docs/REPORT.md`**](docs/REPORT.md): Project notes, sources used, architecture, and live verification report.
- [**`docs/SDK_GUIDE.md`**](docs/SDK_GUIDE.md): Complete developer guide with Go code examples for Bot methods, Mini App components, InitData crypto validation, Updates handling (Long Polling & Webhooks), and Schema Builder usage.
