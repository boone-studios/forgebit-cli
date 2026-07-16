# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

Pre-release — nothing has shipped yet.

### Added

- `forgebit login` / `forgebit logout` — browser-based device-authorization login against a vendor-scoped Forgebit API key
- Multi-vendor profiles — the CLI can hold a stored credential per vendor and switch between them without a fresh login
  - `forgebit vendor list` / `forgebit vendor switch <id|name>`
  - `--vendor <id|name>` on any command to target one vendor for a single call without changing the active default
- `forgebit licenses issue|list|show|verify|revoke|renew` against the Forgebit license API
- `forgebit products list|show|create|update|archive|restore` against the Forgebit products API
- `forgebit webhooks list|show|create|update|delete|rotate-secret|test` and `forgebit webhooks logs list|replay` against the Forgebit webhooks API
- `forgebit licenses public-key` — fetch a vendor's Ed25519 public key
- Fully offline license verification (`forgebit licenses verify --offline`) for `jwt` and `forgebit`-type keys, checked locally with no network call
- `forgebit status` — reports whether the CLI is running against the API or offline data
- `--json` output on `licenses` commands for scripting
- `--version` flag

### Fixed

- Errors were printed twice (once by Cobra, once by our own handler)
- A revoked or expired token now shows a clear message telling you to log in again, instead of a raw 401
- Re-running `forgebit login` for a vendor you're already logged into now revokes the old key instead of leaving it behind
