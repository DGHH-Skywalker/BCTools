# Signed update log

The backend embeds `GoServer/services/update_log_default.json`. On each backend
startup it optionally reads `BctoolData/update-log.json` and
`BctoolData/update-log.sig`. The external JSON is accepted only after its raw
bytes pass Ed25519 verification and its schema, version, timestamps, expiry,
title, and content pass validation. Any failure silently selects the embedded
default.

Only the public key in `GoServer/services/update_log.go` is compiled into the
application. The matching release private seed is kept outside the repository
in an access-controlled release location. Never copy it into the source tree,
application directory, CI logs, or release artifacts. To rotate the key, run
`scripts/generate-update-log-key.cjs` with a secure output path, replace
`updateLogPublicKeyB64`, and preserve the new seed before publishing the build.

After configuring the release public key, sign the exact bytes that will be
copied into `BctoolData`:

```powershell
$env:BCTOOLS_UPDATE_LOG_PRIVATE_KEY = (Get-Content '<secure-key-path>' -Raw).Trim()
go -C GoServer run ./cmd/sign-update-log ../update-log.json ../update-log.sig
Remove-Item Env:BCTOOLS_UPDATE_LOG_PRIVATE_KEY
```

External logs must not be older than the embedded version or publication time,
and their validity window cannot exceed 90 days. Whitespace changes after
signing invalidate the signature. Never commit the private key or place it in
the application/data directory.
