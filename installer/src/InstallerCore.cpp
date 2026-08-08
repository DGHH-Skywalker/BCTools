// InstallerCore.cpp - 安装核心逻辑实现
#include "InstallerCore.h"

#include "AppStrings.h"
#include "Utils.h"
#include "resource.h"
#include "version.h"

#include <shlobj.h>
#include <shlwapi.h>

InstallerCore::InstallerCore(HINSTANCE hInst, const std::wstring& installPath,
                             ProgressCallback callback)
    : hInst_(hInst), installPath_(installPath), callback_(callback) {}

void InstallerCore::Report(int percent, const std::wstring& status) {
    if (callback_) {
        callback_(percent, status);
    }
}

bool InstallerCore::Run() {
    Report(5, L"正在创建安装目录…");
    if (!Utils::EnsureDirectory(installPath_)) {
        lastError_ = L"无法创建安装目录：" + installPath_;
        return false;
    }

    Report(20, L"正在释放文件…");
    if (!ReleaseFiles()) {
        return false;
    }

    Report(60, L"正在创建快捷方式…");
    if (!CreateShortcuts()) {
        return false;
    }

    Report(80, L"正在写入卸载信息…");
    if (!WriteRegistry()) {
        return false;
    }

    Report(100, L"安装完成");
    // 刷新 shell，使快捷方式与卸载注册即时生效（系统“应用”列表立即可见）
    Utils::RefreshShell();
    return true;
}

bool InstallerCore::ReleaseFiles() {
    // 结束可能占用目标目录文件的进程，避免覆盖/复制失败
    Utils::KillProcessByName(L"bctools.exe");
    Utils::KillProcessByName(L"bctool_dev.exe");
    Utils::KillProcessByName(L"uninstall.exe");
    auto pids = Utils::FindPidsByPort(1743);
    for (DWORD pid : pids) {
        Utils::KillProcessByPid(pid);
    }
    Sleep(300);

    // 主程序
    std::wstring mainExe = Utils::JoinPath(installPath_, L"bctools.exe");
    if (!Utils::WriteResourceToFile(hInst_, IDR_BCTOOLS_EXE, mainExe)) {
        // 如果资源未嵌入，尝试从 dist/ 复制
        std::wstring srcMain = Utils::JoinPath(
            Utils::JoinPath(Utils::GetParentDirectory(Utils::GetSelfPath()), L".."),
            L"dist", L"bctools.exe");
        if (!CopyFileW(srcMain.c_str(), mainExe.c_str(), FALSE)) {
            lastError_ = L"无法释放 bctools.exe";
            return false;
        }
    }

    // 开发模式启动器（辅助组件，释放失败不中止主程序安装）
    std::wstring devExe = Utils::JoinPath(installPath_, L"bctool_dev.exe");
    if (!Utils::WriteResourceToFile(hInst_, IDR_BCTOOL_DEV_EXE, devExe)) {
        std::wstring srcDev = Utils::JoinPath(
            Utils::JoinPath(Utils::GetParentDirectory(Utils::GetSelfPath()), L".."),
            L"dist", L"bctool_dev.exe");
        if (!CopyFileW(srcDev.c_str(), devExe.c_str(), FALSE)) {
            // 辅助组件，缺失不影响主程序运行，继续安装
        }
    }

    // 卸载程序：复制自身
    std::wstring uninstallExe = Utils::JoinPath(installPath_, L"uninstall.exe");
    std::wstring selfPath = Utils::GetSelfPath();
    if (!CopyFileW(selfPath.c_str(), uninstallExe.c_str(), FALSE)) {
        lastError_ = L"无法创建 uninstall.exe";
        return false;
    }

    // 写入 uninstall.ini，供卸载程序优先读取目标路径
    std::wstring uninstallIni = Utils::JoinPath(installPath_, L"uninstall.ini");
    WritePrivateProfileStringW(L"Setup", L"InstallPath", installPath_.c_str(),
                               uninstallIni.c_str());

    return true;
}

bool InstallerCore::CreateShortcuts() {
    std::wstring mainExe = Utils::JoinPath(installPath_, L"bctools.exe");
    std::wstring uninstallExe = Utils::JoinPath(installPath_, L"uninstall.exe");
    std::wstring appName(APP_DISPLAY_NAME);
    std::wstring uninstallDesc = std::wstring(L"卸载 ") + APP_DISPLAY_NAME;

    // 桌面快捷方式
    std::wstring desktop = Utils::GetDesktop();
    if (!desktop.empty()) {
        std::wstring desktopLnk = Utils::JoinPath(desktop, appName + L".lnk");
        if (!Utils::CreateShortcut(mainExe, L"", desktopLnk,
                                   APP_DISPLAY_NAME, mainExe, 0)) {
            // 非致命错误，继续
        }
    }

    // 开始菜单快捷方式
    std::wstring startMenu = Utils::GetStartMenuPrograms();
    if (!startMenu.empty()) {
        std::wstring appMenuDir = Utils::JoinPath(startMenu, L"BCTools");
        Utils::EnsureDirectory(appMenuDir);

        std::wstring appLnk = Utils::JoinPath(appMenuDir, appName + L".lnk");
        Utils::CreateShortcut(mainExe, L"", appLnk, APP_DISPLAY_NAME, mainExe, 0);

        std::wstring uninstallLnk = Utils::JoinPath(appMenuDir, L"卸载 " + appName + L".lnk");
        Utils::CreateShortcut(uninstallExe, L"/uninstall", uninstallLnk,
                              uninstallDesc.c_str(), uninstallExe, 0);
    }

    return true;
}

bool InstallerCore::WriteRegistry() {
    bool perMachine = Utils::IsRunAsAdmin();
    std::wstring uninstallExe = Utils::JoinPath(installPath_, L"uninstall.exe");
    std::wstring uninstallCmd = L"\"" + uninstallExe + L"\" /uninstall";
    std::wstring mainExe = Utils::JoinPath(installPath_, L"bctools.exe");

    // 估算大小：主程序 + 开发启动器 + 卸载程序（粗略）
    DWORD sizeKb = 0;
    WIN32_FIND_DATAW fd;
    HANDLE hFind = FindFirstFileW(Utils::JoinPath(installPath_, L"*").c_str(), &fd);
    if (hFind != INVALID_HANDLE_VALUE) {
        do {
            if (!(fd.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY)) {
                sizeKb += (DWORD)((((ULONGLONG)fd.nFileSizeHigh) << 32) |
                                  fd.nFileSizeLow) / 1024;
            }
        } while (FindNextFileW(hFind, &fd));
        FindClose(hFind);
    }

    if (!Utils::WriteUninstallRegistry(perMachine, installPath_, uninstallCmd,
                                       mainExe, APP_VERSION, sizeKb)) {
        lastError_ = L"无法写入卸载注册表项";
        return false;
    }
    return true;
}
