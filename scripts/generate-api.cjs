const path = require("node:path")
const { spawnSync } = require("node:child_process")

const root = path.resolve(__dirname, "..")
const spec = "GoServer/internal/contracts/openapi.json"
const commands = [
  ["go", ["run", "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.6.0",
    "-package", "contracts", "-generate", "types", "-o", "GoServer/internal/contracts/types.gen.go", spec]],
  [process.platform === "win32" ? "npx.cmd" : "npx",
    ["--yes", "openapi-typescript@7.10.1", spec, "-o", "Web/src/api/generated/contracts.ts"]],
]
for (const [command, args] of commands) {
  const result = spawnSync(command, args, { cwd: root, stdio: "inherit", shell: process.platform === "win32" && command === "npx.cmd" })
  if (result.error) throw result.error
  if (result.status !== 0) process.exit(result.status || 1)
}
