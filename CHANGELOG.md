# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- `forgebit licenses expiring` and `forgebit licenses expired` — list licenses expiring in the next N days (default 7) or expired in the last N days (default 30), via `--days`
- `forgebit licenses activations <license-id>` — list a license's active sessions and seat usage
- `forgebit licenses deactivate <activation-id>` — end one session, optionally only if `--client-id` matches
- `forgebit licenses deactivate-all <license-id>` — end every active session for a license
- `forgebit products release <product-id> --file <artifact> --version <x.y.z>` — create a version, upload the artifact, verify its SHA-256, and wait for processing to finish
- `forgebit licenses lookup --product-id <id> --email <email>` — find every license a customer holds for a product by exact email; shows keys when the API key has `download_license`
- `forgebit licenses keys list` and `forgebit licenses keys create` — list the vendor's Ed25519 signing keys or generate a new one (`create` asks for confirmation, since apps verifying offline may need the new public key; `--yes` skips it)

### Changed

- `forgebit status` reports whether credentials and an API URL are configured, instead of calling a placeholder health endpoint that didn't exist
- `--json` is now available on every `licenses`, `products`, and `webhooks` subcommand
- `licenses public-key` no longer requires `--vendor-id`; it defaults to the vendor the CLI is acting as (`--vendor` or the active profile)

### Fixed

- List commands printed a next-page cursor but had no way to use it; `licenses list`, `products list`, `webhooks list`, and `webhooks logs list` now take `--cursor`

### Security

- `products release` no longer prints the presigned upload URL (whose query string is a credential) when the upload fails

### Removed

- The pre-release single-account `token` config field. Credentials live only in vendor profiles; `--token` still works for one-off calls

## [0.1.1] - 2026-07-16

### Fixed

- Homebrew cask now strips the macOS quarantine attribute on install — the unsigned `forgebit` binary was being killed by Gatekeeper (exit 137) on launch

## [0.1.0] - 2026-07-16

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
