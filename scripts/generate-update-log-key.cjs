#!/usr/bin/env node
const fs = require("fs")
const path = require("path")
const { generateKeyPairSync } = require("crypto")

const output = process.argv[2]
if (!output) {
  process.stderr.write("usage: node scripts/generate-update-log-key.cjs <private-key-output>\n")
  process.exit(1)
}

const target = path.resolve(output)
fs.mkdirSync(path.dirname(target), { recursive: true })
const { publicKey, privateKey } = generateKeyPairSync("ed25519")
const publicJwk = publicKey.export({ format: "jwk" })
const privateJwk = privateKey.export({ format: "jwk" })
const publicBase64 = Buffer.from(publicJwk.x, "base64url").toString("base64")
const privateSeedBase64 = Buffer.from(privateJwk.d, "base64url").toString("base64")

fs.writeFileSync(target, `${privateSeedBase64}\n`, { encoding: "utf8", flag: "wx", mode: 0o600 })
process.stdout.write(`${publicBase64}\n`)
