# forgebit-cli

A standalone CLI for [Forgebit](https://forgebit.io) — issue, verify, and manage software licenses from the terminal, with fully offline signature verification for supported license types.

## Install

### Homebrew

```
brew install boone-studios/tap/forgebit
```

### From source

Requires Go 1.26+.

```
git clone git@github.com:boone-studios/forgebit-cli.git
cd forgebit-cli
go build -o forgebit .
```

Move the resulting `forgebit` binary onto your `PATH`.

## Getting started

```
forgebit login
```

Prints a short code, opens your browser to approve the request, and stores a vendor-scoped API key locally (`~/.config/forgebit/config.json` or your OS equivalent). No password ever touches the CLI.

```
forgebit status
```

Reports whether the CLI has credentials and an API URL to talk to, or is running offline.

## Multiple vendors

If you operate more than one vendor account, run `forgebit login` again for each — the CLI stores one credential per vendor and switches instantly with no re-authentication:

```
forgebit vendor list             # see everything you're logged into
forgebit vendor switch <id|name> # change the default for future commands
forgebit licenses list --vendor <id|name>   # target one vendor for a single call
```

`--vendor` never lets the CLI act as a vendor it doesn't already hold a credential for — it only selects among ones you've logged into.

## Licenses

```
forgebit licenses issue --product-id <id> --customer-email you@example.com \
    --tier pro --duration-type trial --license-type jwt

forgebit licenses list [--cursor <next>]
forgebit licenses show <license-id>
forgebit licenses lookup --product-id <id> --email buyer@example.com
forgebit licenses verify <key>
forgebit licenses revoke <license-id> --reason "chargeback"
forgebit licenses renew <license-id> --duration 30_days

forgebit licenses expiring [--days 7]
forgebit licenses expired [--days 30]
```

Every `licenses`, `products`, and `webhooks` command accepts `--json` for scripting. List commands print a `--cursor` value when there are more results.

### Activations

```
forgebit licenses activations <license-id>          # active sessions and seat usage
forgebit licenses deactivate <activation-id>        # end one session
forgebit licenses deactivate-all <license-id>       # end every session
```

### License files and download links

```
forgebit licenses download <license-id> [-o file]   # save the signed license file
forgebit licenses file-link <license-id>            # temporary web link to the license file
forgebit licenses download-link <license-id>        # temporary link to the product ZIP, for purchase emails
forgebit licenses upload-file <license-id> <file>   # attach your own license file instead
forgebit licenses delete-file <license-id>
```

The link commands print only the URL to stdout, so `url=$(forgebit licenses download-link <id>)` works in scripts.

### Offline verification

`jwt` and `forgebit`-type keys are signed with the vendor's Ed25519 key and can be verified with zero network calls — useful for embedding in a product's own license check:

```
forgebit licenses public-key --out vendor.pem
forgebit licenses verify <key> --offline --public-key vendor.pem
```

`--vendor-id` defaults to the vendor the CLI is acting as. Signing keys are managed with:

```
forgebit licenses keys list
forgebit licenses keys create
```

Generating a new key asks for confirmation: apps that verify offline may need its public key, and `public-key` will need `--kid` once more than one key is active.

`serial`, `hmac`, `hwid`, and `encfile`-type keys aren't self-describing and still require the online check.

## Products

```
forgebit products list
forgebit products show <product-id>
forgebit products create --name "My Plugin"
forgebit products update <product-id> --name "My Plugin"
forgebit products archive <product-id>
forgebit products restore <product-id>
```

### Releasing a version

```
forgebit products release <product-id> --file plugin.zip --version 1.2.0 [--channel beta]
```

Creates the version, uploads the file straight to storage, verifies its SHA-256, and waits for Forgebit to finish processing it (`--no-wait` to skip). If the upload fails partway, the version number is already taken, so retry with a new one.

## Webhooks

```
forgebit webhooks list
forgebit webhooks create --url https://example.com/hooks --event license.revoked
forgebit webhooks update <webhook-id> --disabled
forgebit webhooks test <webhook-id>
forgebit webhooks rotate-secret <webhook-id>
forgebit webhooks delete <webhook-id>

forgebit webhooks logs list [--status failed]
forgebit webhooks logs replay <log-id>
```

Signing secrets are shown once, on `create` and `rotate-secret`.

## Development

```
go build ./...
go vet ./...
go test ./...
gofmt -l .
```

## License

[MIT](LICENSE)
