# Signed update log

The backend embeds `GoServer/services/update_log_default.json`. On each backend
startup it optionally reads `BctoolData/update-log.json` and
`BctoolData/update-log.sig`. The external JSON is accepted only after its raw
bytes pass Ed25519 verification and its schema, version, timestamps, expiry,
title, and content pass validation. Any failure silently selects the embedded
default.

Only the public key in `GoServer/services/update_log.go` is compiled into the
application. The randomly generated default public key is fail-closed: its
private counterpart was not stored. Before operating external update logs,
generate a release Ed25519 key offline, replace `updateLogPublicKeyB64` with that
public key, and keep the private key exclusively in an access-controlled release
secret or offline signing device.

After configuring the release public key, sign the exact bytes that will be
copied into `BctoolData`:

```powershell
$env:BCTOOLS_UPDATE_LOG_PRIVATE_KEY = '<base64 seed or private key>'
go -C GoServer run ./cmd/sign-update-log ../update-log.json ../update-log.sig
Remove-Item Env:BCTOOLS_UPDATE_LOG_PRIVATE_KEY
```

Whitespace changes after signing invalidate the signature. Never commit the
private key or place it in the application/data directory.
