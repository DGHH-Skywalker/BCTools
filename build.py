#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""BroadcastTool 一键构建脚本 v5.0.0.1

用法:
  python build.py                完整构建（前端 + 后端 + 打包）
  python build.py --skip-front   跳过前端构建
  python build.py --skip-back    跳过后端编译
  python build.py --clean        清理构建产物
  python build.py --no-mirror    不使用国内镜像源
"""

from __future__ import annotations

import argparse
import os
import platform
import shutil
import subprocess
import sys
from datetime import datetime
from pathlib import Path

ROOT = Path(__file__).resolve().parent
BE = ROOT / "backend"
FE = ROOT / "frontend"
ED = BE / "embed" / "dist"
DIST = ROOT / "dist"
EXE_NAME = "broadcast-tool.exe"


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
        error("frontend/package.json 不存在")
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
    step("复制前端产物到后端 embed/dist")
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


def generate_versioninfo() -> bool:
    """根据 versioninfo.json 生成 Windows 资源文件（可选）"""
    vi = BE / "versioninfo.json"
    syso = BE / "resource.syso"
    if not vi.exists():
        warn("未找到 backend/versioninfo.json，跳过资源文件生成")
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


def build_backend() -> bool:
    step("编译后端 (Go)")
    if not BE.exists():
        error(f"后端目录不存在: {BE}")
        return False
    if not (BE / "main.go").exists():
        error("backend/main.go 不存在")
        return False

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
    exe = ROOT / EXE_NAME
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
    return True


def find_binary(name: str) -> Path | None:
    """按应用目录、exe 同目录、PATH 顺序查找二进制"""
    candidates = [
        ROOT / name,
        BE / name,
        DIST / name,
    ]
    for p in candidates:
        if p.exists():
            return p

    ok_r, out = run(["where", name] if platform.system() == "Windows" else ["which", name], timeout=10, shell=True)
    if ok_r and out:
        p = Path(out.splitlines()[0].strip())
        if p.exists():
            return p
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


def package_dist() -> bool:
    step("打包分发目录 dist/")
    DIST.mkdir(parents=True, exist_ok=True)

    src_exe = ROOT / EXE_NAME
    dst_exe = DIST / EXE_NAME
    if not src_exe.exists():
        error(f"可执行文件不存在: {src_exe}")
        return False
    if not _safe_copy(src_exe, dst_exe):
        warn(f"dist/ 打包跳过 exe 复制，但根目录仍可运行: {src_exe}")

    missing = []
    for name in ("ffmpeg.exe", "ffprobe.exe"):
        src = find_binary(name)
        dst = DIST / name
        if src:
            if _safe_copy(src, dst):
                ok(f"已复制: {dst}")
        else:
            missing.append(name)
            warn(f"未找到 {name}，请手动放置到 {DIST}")

    if missing:
        warn(f"分发包缺少依赖: {', '.join(missing)}")
        warn("运行时需要 ffmpeg.exe / ffprobe.exe 才能进行音频转换")
    else:
        ok("分发包依赖完整")

    sz = sum(f.stat().st_size for f in DIST.rglob("*") if f.is_file()) / (1024 * 1024)
    ok(f"dist/ 打包完成，总大小: {sz:.1f} MB")
    return True


def clean() -> None:
    step("清理构建产物")
    targets = [
        FE / "dist",
        ED,
        ROOT / EXE_NAME,
        DIST,
    ]
    for d in targets:
        if d.exists():
            if d.is_dir():
                shutil.rmtree(d)
            else:
                d.unlink()
            ok(f"删除: {d}")

    for tmp in BE.rglob("*.tmp"):
        try:
            tmp.unlink()
            ok(f"删除临时文件: {tmp}")
        except Exception:
            pass

    ok("清理完成")


def main() -> int:
    parser = argparse.ArgumentParser(
        description="广播站歌单工具 - 一键构建脚本",
        formatter_class=argparse.RawTextHelpFormatter,
    )
    parser.add_argument("--skip-front", action="store_true", help="跳过前端构建")
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
                warn("前端构建失败，使用 backend/embed/dist 中已有产物继续")
            else:
                error("无前端产物可用")
                front_ok = False

    back_ok = True
    if not args.skip_back:
        generate_versioninfo()
        if build_backend():
            ok("后端编译完成")
        else:
            back_ok = False

    dist_ok = True
    if front_ok and back_ok and (ROOT / EXE_NAME).exists():
        if not package_dist():
            dist_ok = False
    elif not args.skip_back:
        warn("跳过 dist/ 打包（可执行文件未生成）")

    print(f"\n{C.B}{bar}{C.X}")
    print(f"{C.B}  构建总结{C.X}")
    print(f"{C.B}{bar}{C.X}")
    results = [
        ("前端构建", front_ok),
        ("后端编译", back_ok),
        ("dist/ 打包", dist_ok),
    ]
    for name, ok_flag in results:
        if ok_flag:
            print(f"  {C.G}[OK]{C.X}  {name}")
        else:
            print(f"  {C.R}[FAIL]{C.X}  {name}")
    print(f"{C.B}{bar}{C.X}\n")

    if front_ok and back_ok and dist_ok:
        info("构建完成，可执行文件位于:")
        print(f"  {DIST / EXE_NAME}")
        print(f"  {ROOT / EXE_NAME}")
        print(f"\n启动方式: 双击 {DIST / EXE_NAME}")
        print(f"本地访问: http://localhost:1743/")
        print(f"数据目录: %APPDATA%\\BroadcastTool\\")
        return 0

    error("构建未完全成功，请检查上方错误日志")
    return 1


if __name__ == "__main__":
    sys.exit(main())
