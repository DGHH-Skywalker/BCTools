#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Build Script for BroadcastTool v5.0.0.1

一键构建前后端：python build.py

参数:
  --skip-front   跳过前端构建
  --skip-back    跳过后端构建
  --clean        清理构建产物
  --no-mirror    不使用国内镜像源
"""

import argparse
import os
import platform
import shutil
import subprocess
import sys
import time
from datetime import datetime
from pathlib import Path

ROOT = Path(__file__).resolve().parent
BE = ROOT / "backend"
FE = ROOT / "frontend"
ED = BE / "embed" / "dist"

_SUPPORTS_ANSI = True
if platform.system() == "Windows":
    try:
        import ctypes
        kernel32 = ctypes.windll.kernel32
        kernel32.SetConsoleMode(kernel32.GetStdHandle(-11), 7)
    except Exception:
        _SUPPORTS_ANSI = False

class C:
    G = "\033[92m" if _SUPPORTS_ANSI else ""
    Y = "\033[93m" if _SUPPORTS_ANSI else ""
    R = "\033[91m" if _SUPPORTS_ANSI else ""
    CYN = "\033[96m" if _SUPPORTS_ANSI else ""
    B = "\033[1m" if _SUPPORTS_ANSI else ""
    X = "\033[0m" if _SUPPORTS_ANSI else ""


def _ts() -> str:
    return datetime.now().strftime("%H:%M:%S")


def info(m): print(f"{C.CYN}[{_ts()} INFO]{C.X} {m}")
def ok(m): print(f"{C.G}[{_ts()}  OK ]{C.X} {m}")
def warn(m): print(f"{C.Y}[{_ts()} WARN]{C.X} {m}")
def error(m): print(f"{C.R}[{_ts()}ERROR]{C.X} {m}")


def step(m):
    bar = "=" * 60
    print(f"\n{C.B}{bar}{C.X}")
    print(f"{C.B}  {_ts()}  {m}{C.X}")
    print(f"{C.B}{bar}{C.X}")


def _run(cmd, timeout=120):
    try:
        if isinstance(cmd, list):
            r = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)
        else:
            r = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout, shell=True)
        return r.returncode == 0, (r.stdout.strip() if r.stdout else r.stderr.strip())
    except FileNotFoundError:
        # Windows 下某些命令（如 npm）需要 shell=True
        if platform.system() == "Windows" and isinstance(cmd, list):
            try:
                import shlex
                cmd_str = subprocess.list2cmdline(cmd)
                r = subprocess.run(cmd_str, capture_output=True, text=True, timeout=timeout, shell=True)
                return r.returncode == 0, (r.stdout.strip() if r.stdout else r.stderr.strip())
            except Exception as e2:
                return False, f"命令失败: {cmd[0]} - {e2}"
        return False, f"未找到命令: {cmd[0] if isinstance(cmd, list) else cmd}"
    except subprocess.TimeoutExpired:
        return False, f"超时 ({timeout}s)"
    except Exception as e:
        return False, str(e)



def check_deps():
    step("检查依赖环境")
    go_ok = node_ok = False
    ok_r, out = _run(["go", "version"], timeout=10)
    if ok_r:
        ok(f"Go: {out}")
        go_ok = True
    else:
        error("未找到 Go，请安装 Go 1.22+")
        error("下载: https://go.dev/dl/")
    ok_r, out = _run(["node", "--version"], timeout=10)
    if ok_r:
        ok(f"Node.js: {out}")
        node_ok = True
    else:
        error("未找到 Node.js，请安装 Node.js 20+")
        error("下载: https://nodejs.org/")
    if node_ok:
        npm_ok, npm_out = _run(["npm", "--version"], timeout=10)
        if npm_ok: ok(f"npm: v{npm_out}")
        else: warn("npm 未找到")
    if not go_ok: error("Go 是编译后端必须的")
    if not node_ok: error("Node.js 是构建前端必须的")
    return go_ok, node_ok

def setup_mirrors():
    step("配置国内镜像源")
    tasks = [
        (["go", "env", "-w", "GOPROXY=https://goproxy.cn,direct"], "Go: goproxy.cn"),
        (["npm", "config", "set", "registry", "https://registry.npmmirror.com"], "npm: npmmirror.com"),
    ]
    for cmd, label in tasks:
        ok_r, _ = _run(cmd, timeout=10)
        if ok_r: ok(label)
        else: warn(f"{label} 设置失败（非致命）")

def build_frontend():
    step("构建前端 (Vue 3 + Vite)")
    if not FE.exists(): error(f"前端目录不存在: {FE}"); return False
    if not (FE / "package.json").exists(): error("package.json 不存在"); return False
    os.chdir(str(FE))
    if not (FE / "node_modules").exists():
        info("安装依赖 (npm install)...")
        ok_r, err = _run(["npm", "install", "--legacy-peer-deps"], timeout=300)
        if not ok_r: error(f"npm install 失败: {err}"); return False
        ok("依赖安装完成")
    else:
        info("node_modules 已存在，跳过 npm install")
    info("类型检查 (npm run type-check)...")
    ok_r, _ = _run(["npm", "run", "type-check"], timeout=60)
    if ok_r: ok("类型检查通过")
    else: warn("类型检查发现问题（非致命，继续构建）")
    info("构建 (npm run build)...")
    ok_r, err = _run(["npm", "run", "build"], timeout=120)
    if not ok_r: error(f"前端构建失败: {err}"); return False
    dist = FE / "dist"
    if dist.exists() and any(dist.iterdir()):
        sz = sum(f.stat().st_size for f in dist.rglob("*") if f.is_file()) / 1024
        ok(f"产物: {dist} ({sz:.0f} KB)")
        return True
    error("产物为空"); return False


def copy_frontend():
    step("复制前端产物到后端")
    d = FE / "dist"
    if not d.exists(): error("前端产物不存在，请先构建"); return False
    if ED.exists(): shutil.rmtree(ED)
    shutil.copytree(str(d), str(ED))
    sz = sum(f.stat().st_size for f in ED.rglob("*") if f.is_file()) / 1024
    ok(f"已复制到: {ED} ({sz:.0f} KB)")
    return True

def build_backend():
    step("编译后端 (Go)")
    if not BE.exists(): error(f"后端目录不存在: {BE}"); return False
    if not (BE / "main.go").exists(): error("main.go 不存在"); return False
    os.chdir(str(BE))
    info("下载依赖 (go mod tidy)...")
    ok_r, err = _run(["go", "mod", "tidy"], timeout=120)
    if not ok_r: error(f"go mod tidy 失败: {err}"); return False
    ok("依赖已同步")
    info("格式化 (go fmt)...")
    ok_r, _ = _run(["go", "fmt", "./..."], timeout=30)
    if ok_r: ok("go fmt 完成")
    else: warn("go fmt 失败（非致命）")
    info("静态检查 (go vet)...")
    ok_r, out = _run(["go", "vet", "./..."], timeout=60)
    if ok_r: ok("go vet 通过")
    else: warn(f"go vet 发现问题:\n{out}")
    info("编译 (go build)...")
    exe = ROOT / "broadcast-tool.exe"
    cmd = ["go", "build", "-ldflags=-s -w -H windowsgui", "-o", str(exe), "."]
    ok_r, err = _run(cmd, timeout=120)
    if not ok_r or not exe.exists():
        error(f"编译失败: {err}")
        return False
    kb = exe.stat().st_size / 1024
    ok(f"成功! {exe} ({kb:.0f} KB)")
    return True


def clean():
    step("清理构建产物")
    for d in [FE / "dist", FE / "node_modules", ED]:
        if d.exists(): shutil.rmtree(d); ok(f"删除: {d}")
    for f in [BE / "broadcast-tool.exe", BE / "go.sum"]:
        if f.exists(): f.unlink(); ok(f"删除: {f}")
    for t in BE.rglob("*.tmp"):
        try: t.unlink(); warn(f"删除临时文件: {t}")
        except: pass
    ok("清理完成!")

class BuildResult:
    def __init__(self):
        self.steps = []
        self._start = time.time()
    def add(self, name, success, detail=""):
        self.steps.append({"name": name, "success": success, "detail": detail})
    def elapsed(self):
        t = time.time() - self._start
        return f"{t:.0f}s" if t < 60 else f"{t//60:.0f}m {t%60:.0f}s"
    def print_summary(self):
        bar = "=" * 60
        print(f"\n{C.B}{bar}{C.X}")
        print(f"{C.B}  构建总结 ({self.elapsed()}){C.X}")
        print(f"{C.B}{bar}{C.X}")
        passed = failed = 0
        for s in self.steps:
            if s["success"]:
                passed += 1
                sym = "\u2713"
                print(f"  {C.G}{sym}{C.X}  {s['name']}")
            else:
                failed += 1
                sym = "\u2717"
                d = f" - {s['detail']}" if s.get("detail") else ""
                print(f"  {C.R}{sym}{C.X}  {s['name']}{d}")
        total = passed + failed
        if failed == 0:
            print(f"\n{C.G}\u2713 {total}/{total} 全部通过{C.X}")
        else:
            print(f"\n{C.R}\u2717 {failed}/{total} 步失败，请检查上方错误{C.X}")
        print(f"{C.B}{bar}{C.X}")

def main():
    parser = argparse.ArgumentParser(
        description="广播站歌单工具 - 一键构建脚本",
        formatter_class=argparse.RawTextHelpFormatter)
    parser.add_argument("--skip-front", action="store_true", help="跳过前端构建与复制")
    parser.add_argument("--skip-back", action="store_true", help="跳过后端编译")
    parser.add_argument("--clean", action="store_true", help="清理构建产物")
    parser.add_argument("--no-mirror", action="store_true", help="不使用国内镜像源")
    args = parser.parse_args()

    bar = "=" * 60
    print(f"{C.CYN}{bar}{C.X}")
    print(f"{C.CYN}  广播站歌单与音频整理工具 - 一键构建{C.X}")
    print(f"{C.CYN}  BroadcastTool Build Script v5.0.0.1{C.X}")
    print(f"{C.CYN}  {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}{C.X}")
    print(f"{C.CYN}{bar}{C.X}\n")

    if args.clean:
        clean()
        return

    if platform.system() != "Windows":
        warn(f"当前系统: {platform.system()}，脚本主要为 Windows 设计")

    result = BuildResult()

    go_ok, node_ok = check_deps()

    if not go_ok and not args.skip_back:
        result.add("检查 Go 环境", False, "缺少 Go")
    if not node_ok and not args.skip_front:
        result.add("检查 Node.js 环境", False, "缺少 Node.js")

    if (not go_ok and not args.skip_back) or (not node_ok and not args.skip_front):
        error("必要依赖缺失，请先安装后重试")
        result.print_summary()
        sys.exit(1)

    if not args.no_mirror and (go_ok or node_ok):
        setup_mirrors()

    front_ok = True
    if not args.skip_front:
        if build_frontend():
            ok("前端构建完成")
            if copy_frontend():
                result.add("前端构建 + 复制", True)
            else:
                result.add("复制前端产物", False, "复制失败")
                front_ok = False
        else:
            warn("前端构建失败，尝试使用已有产物...")
            if ED.exists():
                ok("使用后端 embed 中的已有前端产物")
                result.add("前端构建", False, "构建失败，使用已有产物")
            else:
                error("无前端产物可用")
                result.add("前端构建", False, "构建失败且无备用产物")
                front_ok = False

    back_ok = True
    if not args.skip_back:
        if build_backend():
            result.add("后端编译", True)
        else:
            result.add("后端编译", False, "编译失败")
            back_ok = False

    result.print_summary()

    if front_ok and back_ok:
        exe = ROOT / "broadcast-tool.exe"
        print(f"  构建完成! ({result.elapsed()})")
        print()
        print(f"  启动: 双击 broadcast-tool.exe")
        print(f"        http://localhost:1743/")
        print(f"  数据: %APPDATA%\\BroadcastTool\\")
        print(f"  端口: 1743（占用时自动递增）")
        print(f"  提示: ffmpeg 需在 PATH 或 dist/ 目录中")
    else:
        print(f"\n  构建未完全成功，请检查上方错误日志")
        sys.exit(1)


if __name__ == "__main__":
    main()
