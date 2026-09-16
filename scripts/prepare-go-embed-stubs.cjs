#!/usr/bin/env node
const fs = require("fs")
const path = require("path")

const root = path.resolve(__dirname, "..")
const stubs = [
  ["GoServer/embed/dist/index.html", "<!doctype html><title>BCTools test stub</title>\n"],
  ["GoServer/embed/dist/assets/embed-stub.txt", "test-only embed placeholder\n"],
  ["GoServer/embed/um-react/index.html", "<!doctype html><title>BCTools React test stub</title>\n"],
  ["GoServer/internal/binembed/bin/ffmpeg.exe", ""],
  ["GoServer/internal/binembed/bin/ffprobe.exe", ""],
]

for (const relativePath of [
  "GoServer/embed/dist/assets/.embed-stub",
  "GoServer/internal/binembed/bin/.embed-stub",
]) {
  const obsolete = path.join(root, relativePath)
  if (fs.existsSync(obsolete)) fs.unlinkSync(obsolete)
}

for (const [relativePath, content] of stubs) {
  const target = path.join(root, relativePath)
  if (fs.existsSync(target)) continue
  fs.mkdirSync(path.dirname(target), { recursive: true })
  fs.writeFileSync(target, content, "utf8")
}
