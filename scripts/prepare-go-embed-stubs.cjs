#!/usr/bin/env node
const fs = require("fs")
const path = require("path")

const root = path.resolve(__dirname, "..")
const stubs = [
  ["GoServer/embed/dist/index.html", "<!doctype html><title>BCTools test stub</title>\n"],
  ["GoServer/embed/dist/assets/.embed-stub", "test-only embed placeholder\n"],
  ["GoServer/embed/um-react/index.html", "<!doctype html><title>BCTools React test stub</title>\n"],
  ["GoServer/internal/binembed/bin/.embed-stub", "test-only embed placeholder\n"],
]

for (const [relativePath, content] of stubs) {
  const target = path.join(root, relativePath)
  if (fs.existsSync(target)) continue
  fs.mkdirSync(path.dirname(target), { recursive: true })
  fs.writeFileSync(target, content, "utf8")
}
