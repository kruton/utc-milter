# utc-milter

`utc-milter` is a Postfix milter that normalizes outbound message `Date`
headers to UTC. For example, a message with:

```text
Date: Wed, 03 Jun 2026 12:34:56 -0700
```

is accepted with:

```text
Date: Wed, 03 Jun 2026 19:34:56 +0000
```

If a message has no `Date` header, or the existing `Date` header cannot be
parsed, utc-milter sets it using the current UTC time.

## Build

Go 1.25 or newer is required by the milter library dependency.

```sh
go test ./...
go build ./cmd/utc-milter
```

## Development

This repository uses `mise` for developer tooling and `lefthook` for Git hooks.

```sh
mise install
lefthook install
go test ./...
go vet ./...
go build -o /tmp/utc-milter ./cmd/utc-milter
actionlint
```

The pre-commit hook runs Go tests, `go vet`, a Go build, and `actionlint` for
the GitHub Actions workflow.

## Run

The default listener is a Unix socket for local Postfix integration:

```sh
utc-milter --network unix --socket /run/utc-milter/utc-milter.sock
```

For manual testing, TCP can be used:

```sh
utc-milter --network tcp --socket 127.0.0.1:8899
```

## Postfix

Attach utc-milter only to outbound mail paths, such as submission or an
outbound-only Postfix instance. The daemon rewrites every message it receives.

Example `main.cf` fragment for a local Unix socket:

```text
smtpd_milters = unix:/run/utc-milter/utc-milter.sock
non_smtpd_milters = unix:/run/utc-milter/utc-milter.sock
milter_default_action = accept
```

Use Postfix service-specific `-o smtpd_milters=...` overrides in `master.cf`
when you only want submission or another specific outbound path to use the
milter.

## Alpine APK

The packaging files are in `packaging/alpine`.

```sh
cd packaging/alpine
abuild checksum
abuild -r
```

The APKBUILD uses tagged release tarballs from
`https://github.com/kruton/utc-milter`. The OpenRC subpackage installs
`/etc/init.d/utc-milter` and `/etc/conf.d/utc-milter`.

GitHub Actions signs APKs and repository indexes with the abuild private key in
the `ABUILD_PRIVATE_KEY` repository secret. The matching public key is committed
as `utc-milter.rsa.pub` and is published with the Alpine repository.

## Alpine Repository

Tagged semver releases such as `v0.1.0` publish a signed Alpine repository to
GitHub Pages. As root on the Alpine host, install the repository signing key in
`/etc/apk/keys`, add the architecture-specific repository URL, and update the
APK index:

```sh
install -d -m 0755 /etc/apk/keys
wget -O /etc/apk/keys/utc-milter.rsa.pub \
	https://kruton.github.io/utc-milter/utc-milter.rsa.pub

repo="https://kruton.github.io/utc-milter/$(apk --print-arch)"
grep -qxF "$repo" /etc/apk/repositories || echo "$repo" >> /etc/apk/repositories

apk update
apk add utc-milter utc-milter-openrc
```
