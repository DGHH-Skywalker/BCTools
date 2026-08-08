#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""小播点歌工具一键构建脚本 v5.0.0.1

用法:
  python build.py                完整构建（前端 + um-react + 后端 + 打包）
  python build.py --skip-front   跳过前端构建
  python build.py --skip-back    跳过后端编译
  python build.py --skip-um-react 跳过 um-react 构建
  python build.py --clean        清理构建产物
  python build.py --no-mirror    不使用国内镜像源

构建产物：
  - 单文件可执行程序：ROOT/dist/bctools.exe
  - 开发模式启动器：ROOT/dist/bctool_dev.exe
  - 该 exe 已内含 ffmpeg.exe / ffprobe.exe，首次运行时释放到 %APPDATA%\\BroadcastTool\\bin
"""

from __future__ import annotations

import argparse
import os
import platform
import re
import shutil
import subprocess
import sys
from datetime import datetime
from pathlib import Path

ROOT = Path(__file__).resolve().parent
BE = ROOT / "GoServer"
FE = ROOT / "Web"
UM_REACT = FE / "um-react"
ED = BE / "embed" / "dist"
UM_REACT_ED = BE / "embed" / "um-react"
# 非破坏性注入脚本：在 um-react 解密卡片中加入「导入」按钮，桥接到主应用
UM_REACT_BRIDGE_SRC = FE / "scripts" / "um-react-bridge.js"
# 精简版 ffmpeg/ffprobe 唯一来源（项目内置，不允许回退到外部完整版 ffmpeg）
FFMPEG_MINI_DIST = BE / "ffmpeg_bctools_mini" / "dist"
BIN_EMBED = BE / "internal" / "binembed" / "bin"
DIST = ROOT / "dist"
EXE_NAME = "bctools.exe"
DEV_EXE_NAME = "bctool_dev.exe"
TEST_EXE_NAME = "bctools_test.exe"

INSTALLER = ROOT / "installer"
INSTALLER_RES = INSTALLER / "res"
INSTALLER_SETUP = INSTALLER / "setup.exe"
SETUP_NAME = "BCTools-Setup.exe"


def _supports_ansi() -> bool:
    if platform.system() != "Windows":
        return True
    try:
        import ctypes
        kernel32 = ctypes.windll.kernel32
        kernel32.SetConsoleMode(kernel32.GetStdHandle(-11), 7)
        return True
    except Exception:
        return False


_ANSI = _supports_ansi()


class C:
    G = "\033[92m" if _ANSI else ""
    Y = "\033[93m" if _ANSI else ""
    R = "\033[91m" if _ANSI else ""
    CYN = "\033[96m" if _ANSI else ""
    B = "\033[1m" if _ANSI else ""
    X = "\033[0m" if _ANSI else ""


def _ts() -> str:
    return datetime.now().strftime("%H:%M:%S")


def info(m: str) -> None:
    print(f"{C.CYN}[{_ts()} INFO]{C.X} {m}")


def ok(m: str) -> None:
    print(f"{C.G}[{_ts()}  OK ]{C.X} {m}")


def warn(m: str) -> None:
    print(f"{C.Y}[{_ts()} WARN]{C.X} {m}")


def error(m: str) -> None:
    print(f"{C.R}[{_ts()}ERROR]{C.X} {m}")


def step(m: str) -> None:
    bar = "=" * 60
    print(f"\n{C.B}{bar}{C.X}")
    print(f"{C.B}  {_ts()}  {m}{C.X}")
    print(f"{C.B}{bar}{C.X}")


def run(cmd, cwd: Path | str | None = None, timeout: int = 180, env: dict | None = None, shell: bool = False) -> tuple[bool, str]:
    """运行命令，返回 (是否成功, 输出/错误信息)

    Windows 下若命令未找到且未带扩展名，会尝试 .cmd 后缀（如 npm -> npm.cmd）。
    """
    if isinstance(cmd, list):
        cmd = [str(c) for c in cmd]

    merged_env = os.environ.copy()
    if env:
        merged_env.update(env)

    def _try(cmd_arg, shell_flag: bool) -> tuple[bool, str]:
        try:
            r = subprocess.run(
                cmd_arg,
                cwd=str(cwd) if cwd else None,
                capture_output=True,
                text=True,
                encoding="utf-8",
                errors="replace",
                timeout=timeout,
                env=merged_env,
                shell=shell_flag,
            )
            return r.returncode == 0, (r.stdout.strip() if r.stdout else r.stderr.strip())
        except FileNotFoundError:
            return False, ""
        except subprocess.TimeoutExpired:
            return False, f"超时 ({timeout}s)"
        except Exception as e:
            return False, str(e)

    ok_r, out = _try(cmd, shell)
    if ok_r:
        return True, out

    if out:
        return False, out

    # Windows 下尝试 .cmd 变体
    if platform.system() == "Windows" and isinstance(cmd, list) and not cmd[0].lower().endswith(".cmd"):
        cmd_cmd = [cmd[0] + ".cmd"] + cmd[1:]
        ok_r, out = _try(cmd_cmd, shell)
        if ok_r:
            return True, out
        if out:
            return False, out

    exe = cmd[0] if isinstance(cmd, list) else cmd.split()[0]
    return False, f"未找到命令: {exe}"


def check_deps() -> tuple[bool, bool, bool]:
    """检查 Go / Node / npm 环境"""
    step("检查构建环境")
    go_ok, node_ok, npm_ok = False, False, False

    ok_r, out = run(["go", "version"], timeout=10)
    if ok_r:
        ok(f"Go: {out}")
        go_ok = True
    else:
        error("未找到 Go，请安装 Go 1.22+")

    ok_r, out = run(["node", "--version"], timeout=10)
    if ok_r:
        ok(f"Node.js: {out}")
        node_ok = True
    else:
        error("未找到 Node.js，请安装 Node.js 18+")

    if node_ok:
        ok_r, out = run(["npm", "--version"], timeout=10)
        if ok_r:
            ok(f"npm: v{out}")
            npm_ok = True
        else:
            warn("npm 未找到")

    return go_ok, node_ok, npm_ok


def setup_mirrors() -> None:
    step("配置国内镜像源")
    tasks = [
        (["go", "env", "-w", "GOPROXY=https://goproxy.cn,direct"], "Go: goproxy.cn"),
        (["npm", "config", "set", "registry", "https://registry.npmmirror.com"], "npm: npmmirror.com"),
    ]
    for cmd, label in tasks:
        ok_r, _ = run(cmd, timeout=10)
        if ok_r:
            ok(label)
        else:
            warn(f"{label} 设置失败（非致命）")


def build_frontend() -> bool:
    step("构建前端 (Vue 3 + Vite)")
    if not FE.exists():
        error(f"前端目录不存在: {FE}")
        return False
    if not (FE / "package.json").exists():
        error("Web/package.json 不存在")
        return False

    if not (FE / "node_modules").exists():
        info("安装前端依赖...")
        ok_r, err = run(["npm", "install", "--legacy-peer-deps"], cwd=FE, timeout=300)
        if not ok_r:
            error(f"npm install 失败: {err}")
            return False
        ok("前端依赖安装完成")
    else:
        info("node_modules 已存在，跳过 npm install")

    info("类型检查 (npm run type-check)...")
    ok_r, out = run(["npm", "run", "type-check"], cwd=FE, timeout=120)
    if ok_r:
        ok("类型检查通过")
    else:
        warn(f"类型检查发现问题（非致命，继续构建）:\n{out}")

    info("构建前端 (npm run build)...")
    ok_r, err = run(["npm", "run", "build"], cwd=FE, timeout=180)
    if not ok_r:
        error(f"前端构建失败: {err}")
        return False

    dist = FE / "dist"
    if dist.exists() and any(dist.iterdir()):
        sz = sum(f.stat().st_size for f in dist.rglob("*") if f.is_file()) / 1024
        ok(f"前端产物: {dist} ({sz:.0f} KB)")
        return True

    error("前端产物为空")
    return False


def copy_frontend() -> bool:
    step("复制前端产物到 GoServer/embed/dist")
    src = FE / "dist"
    if not src.exists():
        error("前端产物不存在，请先构建前端")
        return False
    if ED.exists():
        shutil.rmtree(ED)
    shutil.copytree(str(src), str(ED))
    sz = sum(f.stat().st_size for f in ED.rglob("*") if f.is_file()) / 1024
    ok(f"已复制到: {ED} ({sz:.0f} KB)")
    return True


# 解析后的 pnpm 命令：可能是 ["pnpm"] 或 ["corepack", "pnpm"]
PNPM: list[str] = ["pnpm"]
# 需前置到 PATH 的目录（含 corepack 生成的 pnpm shim），使子进程内嵌的 `pnpm` 调用可解析
PNPM_PATH_PREPEND: str | None = None


def _pnpm_env() -> dict | None:
    """返回带 pnpm shim 目录前置到 PATH 的环境变量；无需时返回 None。"""
    if not PNPM_PATH_PREPEND:
        return None
    env = os.environ.copy()
    sep = ";" if platform.system() == "Windows" else ":"
    env["PATH"] = PNPM_PATH_PREPEND + sep + env.get("PATH", "")
    return env


def _ensure_pnpm_shim() -> None:
    """通过 corepack 把 pnpm shim 写入用户可写目录，供子进程内嵌的 `pnpm` 调用解析。

    `corepack enable` 默认写入 Node 安装目录（如 C:\\Program Files\\nodejs），无管理员
    权限会 EPERM。改用 --install-directory 写入项目内可写目录，并记录供 PATH 前置使用。
    """
    global PNPM_PATH_PREPEND
    shim_dir = ROOT / ".corepack-shims"
    try:
        shim_dir.mkdir(parents=True, exist_ok=True)
        run(["corepack", "enable", "--install-directory", str(shim_dir), "pnpm"], timeout=30)
        shim_name = "pnpm.cmd" if platform.system() == "Windows" else "pnpm"
        if (shim_dir / shim_name).exists():
            PNPM_PATH_PREPEND = str(shim_dir)
            ok(f"pnpm shim: {shim_dir}")
        else:
            warn("corepack shim 未生成，子进程内的 pnpm 调用可能失败")
    except Exception as e:
        warn(f"创建 pnpm shim 失败（非致命）: {e}")


def ensure_pnpm() -> bool:
    """确保 pnpm 可用；缺失则尝试通过 corepack 启用。

    corepack 随 Node 附带，但 `corepack enable` 需向 Node 安装目录写入 shim，
    在无管理员权限的 Windows 环境会失败 (EPERM)。因此优先直接用 `corepack pnpm`
    代理调用，并把 shim 写入可写目录前置到 PATH，使构建脚本内嵌的 `pnpm` 调用可解析。
    """
    global PNPM
    # 1) 直接 pnpm
    ok_r, out = run(["pnpm", "--version"], timeout=15)
    if ok_r:
        ok(f"pnpm: v{out}")
        PNPM = ["pnpm"]
        return True
    # 2) corepack pnpm 代理（无需写入 Program Files）
    ok_r, out = run(["corepack", "pnpm", "--version"], timeout=30)
    if not ok_r:
        # 3) 尝试 corepack enable + prepare 后再用 corepack pnpm
        info("pnpm 未找到，尝试通过 corepack 启用...")
        run(["corepack", "enable"], timeout=30)
        run(["corepack", "prepare", "pnpm@10.15.1", "--activate"], timeout=60)
        ok_r, out = run(["corepack", "pnpm", "--version"], timeout=30)
    if ok_r:
        ok(f"pnpm (corepack): v{out}")
        PNPM = ["corepack", "pnpm"]
        _ensure_pnpm_shim()
        return True
    error("未找到 pnpm，且 corepack 启用失败。请手动安装 pnpm（npm i -g pnpm）后重试。")
    return False


def patch_um_react_base(dist: Path) -> None:
    """把构建产物 index.html 的路由 base 改为 /um-react/（仅改产物，不动子模块源码）。"""
    idx = dist / "index.html"
    try:
        text = idx.read_text(encoding="utf-8")
        new = text.replace("window.BASE_URL = '/'", "window.BASE_URL = '/um-react/'")
        if new == text:
            warn("未在 um-react index.html 找到 window.BASE_URL，路由 base 未修改")
        else:
            idx.write_text(new, encoding="utf-8")
            ok("已设置 um-react 路由 base 为 /um-react/")
    except Exception as e:
        warn(f"修改 um-react index.html 失败（非致命）: {e}")


def inject_um_react_bridge(dist: Path) -> None:
    """把桥接脚本复制进 um-react 产物，并在 index.html 注入 <script> 引用（非破坏性）。

    桥接脚本在运行时于解密卡片注入「导入」按钮，把音频上传到后端暂存
    (POST /api/decrypt/stage)，再跳转到主应用导入页完成导入。跨页面通信走后端，
    便于局域网内不同设备（含手机）访问。不修改 um-react 源码，便于独立更新。
    """
    if not UM_REACT_BRIDGE_SRC.exists():
        warn(f"桥接脚本不存在，跳过注入: {UM_REACT_BRIDGE_SRC}")
        return
    try:
        shutil.copyfile(str(UM_REACT_BRIDGE_SRC), str(dist / "um-react-bridge.js"))
        idx = dist / "index.html"
        text = idx.read_text(encoding="utf-8")
        tag = '<script src="./um-react-bridge.js"></script>'
        if tag not in text:
            # 注入到 <head> 末尾，早于 React bundle 执行以便尽早接管 DOM
            new = text.replace("</head>", tag + "</head>", 1)
            if new == text:
                new = text + tag
            idx.write_text(new, encoding="utf-8")
        ok("已注入 um-react 桥接脚本")
    except Exception as e:
        warn(f"注入 um-react 桥接脚本失败（非致命）: {e}")


def build_um_react() -> bool:
    step("构建 um-react (Unlock Music)")
    if not UM_REACT.exists() or not (UM_REACT / "package.json").exists():
        error(f"um-react 子模块不存在: {UM_REACT}（尝试 git submodule update --init --recursive）")
        return False

    if not ensure_pnpm():
        return False

    if not (UM_REACT / "node_modules").exists():
        info("安装 um-react 依赖 (pnpm install)...")
        ok_r, err = run([*PNPM, "install", "--no-frozen-lockfile"], cwd=UM_REACT, timeout=600, env=_pnpm_env())
        if not ok_r:
            # simple-git-hooks 的 prepare 脚本在子模块里可能报错，忽略脚本重试并重建原生依赖
            warn(f"pnpm install 失败，尝试 --ignore-scripts 后重建原生依赖:\n{err}")
            ok_r, err = run([*PNPM, "install", "--ignore-scripts"], cwd=UM_REACT, timeout=600, env=_pnpm_env())
            if not ok_r:
                error(f"pnpm install 失败: {err}")
                return False
            ok_r2, _ = run([*PNPM, "rebuild"], cwd=UM_REACT, timeout=300, env=_pnpm_env())
            if not ok_r2:
                warn("pnpm rebuild 存在问题（可能影响构建）")
        ok("um-react 依赖安装完成")
    else:
        info("um-react node_modules 已存在，跳过 install")

    info("构建 um-react (pnpm build)...")
    ok_r, err = run([*PNPM, "build"], cwd=UM_REACT, timeout=300, env=_pnpm_env())
    if not ok_r:
        error(f"um-react 构建失败: {err}")
        return False

    dist = UM_REACT / "dist"
    if not dist.exists() or not (dist / "index.html").exists():
        error("um-react 产物为空")
        return False

    patch_um_react_base(dist)
    inject_um_react_bridge(dist)
    sz = sum(f.stat().st_size for f in dist.rglob("*") if f.is_file()) / 1024
    ok(f"um-react 产物: {dist} ({sz:.0f} KB)")
    return True


def copy_um_react() -> bool:
    step("复制 um-react 产物到 GoServer/embed/um-react")
    src = UM_REACT / "dist"
    if not src.exists():
        error("um-react 产物不存在")
        return False
    if UM_REACT_ED.exists():
        shutil.rmtree(UM_REACT_ED)
    shutil.copytree(str(src), str(UM_REACT_ED))
    sz = sum(f.stat().st_size for f in UM_REACT_ED.rglob("*") if f.is_file()) / 1024
    ok(f"已复制到: {UM_REACT_ED} ({sz:.0f} KB)")
    return True


def ensure_um_react_embed() -> None:
    """确保 embed/um-react 存在（含占位 index.html），避免 go:embed 失败。"""
    UM_REACT_ED.mkdir(parents=True, exist_ok=True)
    if not (UM_REACT_ED / "index.html").exists():
        (UM_REACT_ED / "index.html").write_text(
            "<!doctype html><meta charset=utf-8><title>um-react</title>"
            "<p>um-react 尚未构建，请运行 python build.py（不要带 --skip-um-react）。</p>",
            encoding="utf-8",
        )
        warn(f"已创建占位 {UM_REACT_ED}/index.html（um-react 未构建）")


def generate_versioninfo() -> bool:
    """根据 versioninfo.json 生成 Windows 资源文件（可选）"""
    vi = BE / "versioninfo.json"
    syso = BE / "resource.syso"
    if not vi.exists():
        warn("未找到 GoServer/versioninfo.json，跳过资源文件生成")
        return True

    ok_r, _ = run(["goversioninfo", "--help"], timeout=10)
    if not ok_r:
        warn("未找到 goversioninfo，将使用已有 resource.syso 或默认图标")
        return True

    info("生成 Windows 资源文件 (goversioninfo)...")
    ok_r, err = run(
        ["goversioninfo", "-64", "-o", str(syso), str(vi)],
        cwd=BE,
        timeout=60,
    )
    if ok_r:
        ok(f"已生成: {syso}")
        return True
    warn(f"goversioninfo 生成失败（非致命）: {err}")
    return True


def copy_embedded_binaries() -> bool:
    """把项目内置的精简版 ffmpeg/ffprobe 复制到 Go embed 目录，打包进单个 exe。

    只允许从 GoServer/ffmpeg_bctools_mini/dist 读取，禁止回退到外部完整版 ffmpeg。
    若内置精简版缺失则视为构建错误并直接中止，避免运行时悄悄走系统 PATH 引入非精简版。
    """
    step("嵌入 ffmpeg/ffprobe 到单文件 exe（仅使用项目内置精简版）")
    if not FFMPEG_MINI_DIST.exists():
        error(f"未找到精简版 ffmpeg 目录: {FFMPEG_MINI_DIST}")
        error("请在 GoServer/ffmpeg_bctools_mini 下执行 build.sh 后再构建，"
              "或从已有构建拷贝 dist/ffmpeg.exe / dist/ffprobe.exe 到该目录。")
        return False

    BIN_EMBED.mkdir(parents=True, exist_ok=True)

    missing = []
    for name in ("ffmpeg.exe", "ffprobe.exe"):
        src = FFMPEG_MINI_DIST / name
        dst = BIN_EMBED / name
        if not src.exists():
            missing.append(name)
            warn(f"未在精简版 dist 中找到 {name}")
            continue
        if _safe_copy(src, dst):
            ok(f"已嵌入精简版: {dst} ({dst.stat().st_size / 1024:.0f} KB)")

    if missing:
        error("精简版 ffmpeg/ffprobe 不完整，构建中止（避免运行时回退到系统完整版）。")
        return False
    return True


def build_backend() -> bool:
    step("编译后端 (Go)")
    if not BE.exists():
        error(f"后端目录不存在: {BE}")
        return False
    if not (BE / "main.go").exists():
        error("GoServer/main.go 不存在")
        return False

    copy_embedded_binaries()

    info("同步 Go 依赖 (go mod tidy)...")
    ok_r, err = run(["go", "mod", "tidy"], cwd=BE, timeout=120)
    if not ok_r:
        error(f"go mod tidy 失败: {err}")
        return False
    ok("Go 依赖已同步")

    info("格式化 (go fmt)...")
    ok_r, _ = run(["go", "fmt", "./..."], cwd=BE, timeout=30)
    if ok_r:
        ok("go fmt 完成")
    else:
        warn("go fmt 失败（非致命）")

    info("静态检查 (go vet)...")
    ok_r, out = run(["go", "vet", "./..."], cwd=BE, timeout=60)
    if ok_r:
        ok("go vet 通过")
    else:
        warn(f"go vet 发现问题:\n{out}")

    info("运行测试 (go test)...")
    ok_r, out = run(["go", "test", "./..."], cwd=BE, timeout=120)
    if ok_r:
        ok("go test 通过")
    else:
        warn(f"go test 存在问题:\n{out}")

    info("编译可执行文件 (go build)...")
    exe = DIST / EXE_NAME
    build_env = {"CGO_ENABLED": "0", "GOOS": "windows"}
    cmd = [
        "go", "build",
        "-ldflags=-s -w -H windowsgui",
        "-o", str(exe), ".",
    ]
    ok_r, err = run(cmd, cwd=BE, env=build_env, timeout=180)
    if not ok_r or not exe.exists():
        error(f"后端编译失败: {err}")
        return False

    kb = exe.stat().st_size / 1024
    ok(f"编译成功: {exe} ({kb:.0f} KB)")

    info("编译开发模式启动器 (bctool_dev.exe)...")
    dev_exe = DIST / DEV_EXE_NAME
    dev_cmd = [
        "go", "build",
        "-ldflags=-s -w -H windowsgui",
        "-o", str(dev_exe), "./cmd/devlauncher",
    ]
    ok_r, err = run(dev_cmd, cwd=BE, env=build_env, timeout=120)
    if not ok_r or not dev_exe.exists():
        warn(f"开发模式启动器编译失败: {err}")
    else:
        ok(f"编译成功: {dev_exe} ({dev_exe.stat().st_size / 1024:.0f} KB)")

    return True


def build_test_exe() -> bool:
    """构建一次性测试版 exe：web 服务用 712 端口，数据写 %TEMP% 临时目录，退出即清不留痕迹。

    复用正式版已嵌入的前端/ffmpeg/um-react 资源，仅通过 ldflags 注入端口与无痕模式。
    """
    step("构建一次性测试版 (bctools_test.exe)")
    if not (DIST / EXE_NAME).exists():
        error("正式版 exe 不存在；测试版复用其嵌入资源，请先完成后端编译")
        return False
    test_exe = DIST / TEST_EXE_NAME
    build_env = {"CGO_ENABLED": "0", "GOOS": "windows"}
    ldflags = "-s -w -H windowsgui -X main.portStr=712 -X main.ephemeralStr=true"
    cmd = ["go", "build", f"-ldflags={ldflags}", "-o", str(test_exe), "."]
    ok_r, err = run(cmd, cwd=BE, env=build_env, timeout=180)
    if not ok_r or not test_exe.exists():
        error(f"测试版编译失败: {err}")
        return False
    ok(f"编译成功: {test_exe} ({test_exe.stat().st_size / 1024:.0f} KB)")
    return True


def find_binary(name: str) -> Path | None:
    """仅在项目内置精简版 ffmpeg 目录中查找二进制；不再回退到外部 PATH。

    保留该函数仅为兼容旧调用方路径查找逻辑，但严格只读项目内置 dist，
    避免引入系统完整版 ffmpeg。
    """
    src = FFMPEG_MINI_DIST / name
    if src.exists():
        return src
    return None


def _safe_copy(src: Path, dst: Path) -> bool:
    """安全复制文件，若目标被占用则先尝试删除。"""
    try:
        if dst.exists():
            try:
                dst.unlink()
            except PermissionError:
                warn(f"目标文件被占用，尝试覆盖: {dst}")
        shutil.copy2(str(src), str(dst))
        return True
    except PermissionError as e:
        error(f"无法复制 {src} -> {dst}: {e}")
        return False
    except Exception as e:
        error(f"复制失败 {src} -> {dst}: {e}")
        return False


def generate_installer_version_header() -> bool:
    """读取 Go 后端版本，生成安装程序使用的 C++ 版本头。"""
    version_file = BE / "internal" / "version" / "version.go"
    header_file = INSTALLER / "src" / "version.h"
    if not version_file.exists():
        warn(f"版本源文件不存在: {version_file}")
        return False
    content = version_file.read_text(encoding="utf-8")
    m = re.search(r'const\s+Version\s+=\s+"([^"]+)"', content)
    if not m:
        warn(f"无法从 {version_file} 解析版本号")
        return False
    version = m.group(1)
    header_file.write_text(
        f'#pragma once\n\n#define APP_VERSION L"{version}"\n',
        encoding="utf-8",
    )
    ok(f"生成安装程序版本头: {header_file} -> {version}")
    return True


def prepare_installer_resources() -> bool:
    """把安装程序所需资源复制到 installer/res/。"""
    INSTALLER_RES.mkdir(parents=True, exist_ok=True)
    resources = [
        (BE / "assets" / "icon-bc.ico", INSTALLER_RES / "icon-bc.ico"),
        (FE / "public" / "logo.png", INSTALLER_RES / "logo.png"),
        # 使用构建时已按项目用字精简后的字体，减小安装包体积
        (FE / "dist" / "fonts" / "江西拙楷3.0.ttf", INSTALLER_RES / "font_jiangxi_zhuokai.ttf"),
        (FE / "dist" / "fonts" / "方正颜宋简体.ttf", INSTALLER_RES / "font_fangzheng_yansong.ttf"),
        (DIST / EXE_NAME, INSTALLER_RES / "bctools.exe"),
        (DIST / DEV_EXE_NAME, INSTALLER_RES / "bctool_dev.exe"),
    ]
    ok_all = True
    for src, dst in resources:
        if not src.exists():
            warn(f"安装程序资源缺失: {src}")
            ok_all = False
            continue
        if _safe_copy(src, dst):
            ok(f"安装程序资源: {dst.name}")
        else:
            ok_all = False
    return ok_all


def _ensure_mingw_on_path() -> bool:
    """若 PATH 中找不到 g++，尝试把常见 MinGW 路径临时加入 PATH。"""
    ok_r, _ = run(["g++", "--version"], timeout=10)
    if ok_r:
        return True

    candidates = [
        Path(r"C:\Tools\mingw64\mingw64\bin"),
        Path(r"C:\mingw64\bin"),
        Path(r"C:\msys64\mingw64\bin"),
        Path(r"C:\Program Files\mingw-w64\x86_64-8.1.0-posix-seh-rt_v6-rev0\mingw64\bin"),
    ]
    for p in candidates:
        if (p / "g++.exe").exists():
            sep = ";" if platform.system() == "Windows" else ":"
            os.environ["PATH"] = str(p) + sep + os.environ.get("PATH", "")
            ok_r, _ = run(["g++", "--version"], timeout=10)
            if ok_r:
                ok(f"自动加载 MinGW: {p}")
                return True
    return False


def build_installer() -> bool:
    """使用 MinGW-w64 编译 C++ 安装程序，产物复制到 dist/。"""
    step("构建安装程序 (C++ Win32/GDI+)")

    if not prepare_installer_resources():
        error("安装程序资源准备失败，跳过安装程序构建")
        return False

    if not generate_installer_version_header():
        warn("生成安装程序版本头失败，将使用代码中硬编码版本")

    if not _ensure_mingw_on_path():
        error("未找到 g++，尝试安装 MinGW-w64 并将 bin 加入 PATH")
        return False

    # 优先尝试 make；Windows 上 MinGW 也常提供 mingw32-make
    make_cmd = None
    for cmd in (["make"], ["mingw32-make"]):
        ok_r, out = run(cmd + ["--version"], cwd=INSTALLER, timeout=10)
        if ok_r:
            make_cmd = cmd
            break
    if not make_cmd:
        error("未找到 make 或 mingw32-make，请安装 MinGW-w64 并将 bin 加入 PATH")
        return False

    ok_r, err = run(make_cmd + ["clean"], cwd=INSTALLER, timeout=30)
    if not ok_r:
        warn(f"清理旧安装程序产物失败（非致命）: {err}")

    ok_r, err = run(make_cmd, cwd=INSTALLER, timeout=180)
    if not ok_r:
        error(f"安装程序编译失败:\n{err}")
        return False

    if not INSTALLER_SETUP.exists():
        error(f"安装程序产物不存在: {INSTALLER_SETUP}")
        return False

    DIST.mkdir(parents=True, exist_ok=True)
    setup_dst = DIST / SETUP_NAME
    if _safe_copy(INSTALLER_SETUP, setup_dst):
        ok(f"安装程序已输出: {setup_dst} ({setup_dst.stat().st_size / 1024:.0f} KB)")
        return True
    return False


def package_dist() -> bool:
    step("整理单文件分发目录 dist/")
    DIST.mkdir(parents=True, exist_ok=True)

    src_exe = DIST / EXE_NAME
    if not src_exe.exists():
        error(f"可执行文件不存在: {src_exe}")
        return False
    ok(f"单文件可执行程序已输出到: {src_exe}")

    dev_exe = DIST / DEV_EXE_NAME
    if dev_exe.exists():
        ok(f"开发模式启动器已输出到: {dev_exe}")

    # 清理旧版可执行文件名，避免用户混淆
    for old_name in ("broadcast-tool.exe", "broadcast-tool.exe~"):
        old = DIST / old_name
        if old.exists():
            try:
                old.unlink()
                info(f"已清理旧文件: {old}")
            except Exception as e:
                warn(f"无法删除 {old}: {e}")

    # ffmpeg/ffprobe 已嵌入 exe，无需额外复制
    for name in ("ffmpeg.exe", "ffprobe.exe"):
        old = DIST / name
        if old.exists():
            try:
                old.unlink()
                info(f"已清理旧文件: {old}")
            except Exception as e:
                warn(f"无法删除 {old}: {e}")

    sz = sum(f.stat().st_size for f in DIST.rglob("*") if f.is_file()) / (1024 * 1024)
    ok(f"dist/ 打包完成，总大小: {sz:.1f} MB（单文件 exe，已内含 ffmpeg/ffprobe）")
    return True


def clean() -> None:
    step("清理构建产物")
    targets = [
        FE / "dist",
        ED,
        UM_REACT_ED,
        BIN_EMBED,
        DIST,
        INSTALLER / "obj",
        INSTALLER_SETUP,
    ]
    for d in targets:
        if d.exists():
            if d.is_dir():
                shutil.rmtree(d)
            else:
                d.unlink()
            ok(f"删除: {d}")

    # 安装程序资源目录中复制的 exe 也一并清理
    for name in ("bctools.exe", "bctool_dev.exe"):
        res_exe = INSTALLER_RES / name
        if res_exe.exists():
            try:
                res_exe.unlink()
                ok(f"删除: {res_exe}")
            except Exception:
                pass

    for tmp in BE.rglob("*.tmp"):
        try:
            tmp.unlink()
            ok(f"删除临时文件: {tmp}")
        except Exception:
            pass

    ok("清理完成")


def main() -> int:
    parser = argparse.ArgumentParser(
        description="小播点歌工具 - 一键构建脚本",
        formatter_class=argparse.RawTextHelpFormatter,
    )
    parser.add_argument("--skip-front", action="store_true", help="跳过前端构建")
    parser.add_argument("--skip-back", action="store_true", help="跳过后端编译")
    parser.add_argument("--skip-um-react", action="store_true", help="跳过 um-react 构建")
    parser.add_argument("--skip-installer", action="store_true", help="跳过安装程序构建")
    parser.add_argument("--skip-test-exe", action="store_true", help="跳过一次性测试版构建")
    parser.add_argument("--clean", action="store_true", help="清理构建产物")
    parser.add_argument("--no-mirror", action="store_true", help="不使用国内镜像源")
    args = parser.parse_args()

    bar = "=" * 60
    print(f"{C.CYN}{bar}{C.X}")
    print(f"{C.CYN}  小播点歌工具 - 一键构建{C.X}")
    print(f"{C.CYN}  XiaoBo Song Request Tool Build Script v5.0.0.1{C.X}")
    print(f"{C.CYN}  {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}{C.X}")
    print(f"{C.CYN}{bar}{C.X}\n")

    if args.clean:
        clean()
        return 0

    if platform.system() != "Windows":
        warn(f"当前系统: {platform.system()}，脚本主要为 Windows 设计")

    go_ok, node_ok, npm_ok = check_deps()
    if (not go_ok and not args.skip_back) or (not node_ok and not args.skip_front):
        error("必要依赖缺失，请先安装后重试")
        return 1

    if not args.no_mirror and (go_ok or node_ok):
        setup_mirrors()

    front_ok = True
    if not args.skip_front:
        if build_frontend() and copy_frontend():
            ok("前端构建与复制完成")
        else:
            if ED.exists():
                warn("前端构建失败，使用 GoServer/embed/dist 中已有产物继续")
            else:
                error("无前端产物可用")
                front_ok = False

    # um-react（独立的音乐解锁工具，作为子模块引入，按需构建并嵌入）
    um_ok = True
    if not args.skip_front and not args.skip_um_react:
        if build_um_react() and copy_um_react():
            ok("um-react 构建与复制完成")
        else:
            um_ok = False
            warn("um-react 构建失败，使用占位产物继续后端编译")
    elif args.skip_um_react:
        info("跳过 um-react 构建（--skip-um-react）")
    ensure_um_react_embed()

    back_ok = True
    if not args.skip_back:
        generate_versioninfo()
        if build_backend():
            ok("后端编译完成")
        else:
            back_ok = False

    dist_ok = True
    if front_ok and back_ok and (DIST / EXE_NAME).exists():
        if not package_dist():
            dist_ok = False
    elif not args.skip_back:
        warn("跳过 dist/ 打包（可执行文件未生成）")

    test_ok = True
    if not args.skip_test_exe and back_ok:
        if build_test_exe():
            ok("测试版构建完成")
        else:
            test_ok = False
    elif args.skip_test_exe:
        info("跳过一次性测试版构建（--skip-test-exe）")

    installer_ok = False
    if not args.skip_installer:
        if dist_ok and (DIST / EXE_NAME).exists() and (DIST / DEV_EXE_NAME).exists():
            if build_installer():
                installer_ok = True
            else:
                warn("安装程序构建失败")
        else:
            warn("主程序未生成，跳过安装程序构建")
    else:
        info("跳过安装程序构建（--skip-installer）")

    print(f"\n{C.B}{bar}{C.X}")
    print(f"{C.B}  构建总结{C.X}")
    print(f"{C.B}{bar}{C.X}")
    results = [
        ("前端构建", front_ok),
        ("um-react 构建", um_ok),
        ("后端编译", back_ok),
        ("dist/ 打包", dist_ok),
        ("测试版 exe", test_ok),
        ("安装程序", installer_ok),
    ]
    for name, ok_flag in results:
        if ok_flag:
            print(f"  {C.G}[OK]{C.X}  {name}")
        else:
            print(f"  {C.R}[FAIL]{C.X}  {name}")
    print(f"{C.B}{bar}{C.X}\n")

    if front_ok and back_ok and dist_ok:
        info("构建完成，产物位于:")
        print(f"  {DIST / EXE_NAME}")
        if (DIST / DEV_EXE_NAME).exists():
            print(f"  {DIST / DEV_EXE_NAME}")
        if test_ok and (DIST / TEST_EXE_NAME).exists():
            print(f"  {DIST / TEST_EXE_NAME}")
        if installer_ok and (DIST / SETUP_NAME).exists():
            print(f"  {DIST / SETUP_NAME}")
        print(f"\n启动方式: 双击 {DIST / EXE_NAME}")
        print(f"开发模式: 双击 {DIST / DEV_EXE_NAME}")
        print(f"本地访问: http://localhost:1743/")
        print(f"数据目录: %APPDATA%\\BroadcastTool\\")
        if test_ok and (DIST / TEST_EXE_NAME).exists():
            print(f"\n测试版: 双击 {DIST / TEST_EXE_NAME}（端口 712，退出不留痕迹）")
        if installer_ok:
            print(f"安装程序: 双击 {DIST / SETUP_NAME}")
        return 0

    error("构建未完全成功，请检查上方错误日志")
    return 1


if __name__ == "__main__":
    sys.exit(main())
