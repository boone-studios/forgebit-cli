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

Reports whether the CLI is authenticated against the live API or running offline.

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

forgebit licenses list
forgebit licenses show <license-id>
forgebit licenses verify <key>
forgebit licenses revoke <license-id> --reason "chargeback"
forgebit licenses renew <license-id> --duration 30_days
```

Every `licenses` command accepts `--json` for scripting.

### Offline verification

`jwt` and `forgebit`-type keys are signed with the vendor's Ed25519 key and can be verified with zero network calls — useful for embedding in a product's own license check:

```
forgebit licenses public-key --vendor-id <id> --out vendor.pem
forgebit licenses verify <key> --offline --public-key vendor.pem
```

`serial`, `hmac`, `hwid`, and `encfile`-type keys aren't self-describing and still require the online check.

## Development

```
go build ./...
go vet ./...
go test ./...
gofmt -l .
```

## License

[MIT](LICENSE)
