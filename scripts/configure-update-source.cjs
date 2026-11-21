const fs = require("node:fs")
const path = require("node:path")

const repository = process.env.BCTOOLS_UPDATE_REPO
const rootJSON = process.env.BCTOOLS_TUF_ROOT_JSON
if (!repository && !rootJSON) process.exit(0)
if (!repository || !/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/.test(repository) || !rootJSON) {
  throw new Error("BCTOOLS_UPDATE_REPO and BCTOOLS_TUF_ROOT_JSON must both be configured")
}
const root = JSON.parse(rootJSON)
if (root.signed?._type !== "root" || !root.signatures?.length) throw new Error("Invalid TUF root")
const branch = process.env.BCTOOLS_UPDATE_BRANCH || "update-feed"
if (!/^[A-Za-z0-9_.-]+$/.test(branch)) throw new Error("Invalid update branch")
const mirrors = JSON.parse(process.env.BCTOOLS_UPDATE_MIRRORS || "[]")
if (!Array.isArray(mirrors) || mirrors.some(mirror => typeof mirror !== "string" || !mirror.startsWith("https://"))) {
  throw new Error("BCTOOLS_UPDATE_MIRRORS must be a JSON array of HTTPS mirror base URLs")
}
const target = path.resolve(__dirname, "../GoServer/internal/updates/source.json")
fs.writeFileSync(target, JSON.stringify({ repository, branch, root, mirrors }, null, 2) + "\n")
