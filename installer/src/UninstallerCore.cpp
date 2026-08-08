// UninstallerCore.cpp - 卸载核心逻辑实现
#include "UninstallerCore.h"

#include "Utils.h"

#include <shlobj.h>
#include <shlwapi.h>
#include <windows.h>

namespace {

static bool ClearAttributesRecursive(const std::wstring& path) {
    DWORD attr = GetFileAttributesW(path.c_str());
    if (attr == INVALID_FILE_ATTRIBUTES) return true;
    if (attr & (FILE_ATTRIBUTE_READONLY | FILE_ATTRIBUTE_HIDDEN | FILE_ATTRIBUTE_SYSTEM)) {
        SetFileAttributesW(path.c_str(), FILE_ATTRIBUTE_NORMAL);
    }
    if (attr & FILE_ATTRIBUTE_DIRECTORY) {
        std::wstring search = Utils::JoinPath(path, L"*");
        WIN32_FIND_DATAW fd;
        HANDLE hFind = FindFirstFileW(search.c_str(), &fd);
        if (hFind != INVALID_HANDLE_VALUE) {
            do {
                std::wstring name = fd.cFileName;
                if (name == L"." || name == L"..") continue;
                ClearAttributesRecursive(Utils::JoinPath(path, name));
            } while (FindNextFileW(hFind, &fd));
            FindClose(hFind);
        }
    }
    return true;
}

static bool DeleteRecursiveInternal(const std::wstring& path) {
    DWORD attr = GetFileAttributesW(path.c_str());
    if (attr == INVALID_FILE_ATTRIBUTES) return true;
    if (!(attr & FILE_ATTRIBUTE_DIRECTORY)) {
        SetFileAttributesW(path.c_str(), FILE_ATTRIBUTE_NORMAL);
        return DeleteFileW(path.c_str()) != 0;
    }
    ClearAttributesRecursive(path);
    std::wstring search = Utils::JoinPath(path, L"*");
    WIN32_FIND_DATAW fd;
    HANDLE hFind = FindFirstFileW(search.c_str(), &fd);
    if (hFind == INVALID_HANDLE_VALUE) {
        return RemoveDirectoryW(path.c_str()) != 0;
    }
    do {
        std::wstring name = fd.cFileName;
        if (name == L"." || name == L"..") continue;
        std::wstring full = Utils::JoinPath(path, name);
        if (fd.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY) {
            DeleteRecursiveInternal(full);
        } else {
            SetFileAttributesW(full.c_str(), FILE_ATTRIBUTE_NORMAL);
            DeleteFileW(full.c_str());
        }
    } while (FindNextFileW(hFind, &fd));
    FindClose(hFind);
    return RemoveDirectoryW(path.c_str()) != 0;
}

static void ScheduleSelfDelete(const std::wstring& path) {
    // 启动一个独立的 cmd 进程，每隔约 1 秒尝试删除自身，最多 30 秒。
    // 循环可以覆盖 UI 关闭前的等待时间，避免文件仍被占用导致删除失败。
    std::wstring quoted = L"\"" + path + L"\"";
    std::wstring cmd =
        L"/C for /L %i in (1,1,30) do @ping 127.0.0.1 -n 2 >nul && if exist " +
        quoted + L" del " + quoted;
    SHELLEXECUTEINFOW sei = {sizeof(sei)};
    sei.lpVerb = L"open";
    sei.lpFile = L"cmd.exe";
    sei.lpParameters = cmd.c_str();
    sei.nShow = SW_HIDE;
    sei.fMask = SEE_MASK_NOASYNC | SEE_MASK_NOCLOSEPROCESS;
    ShellExecuteExW(&sei);
}

} // namespace

UninstallerCore::UninstallerCore(StatusCallback callback)
    : callback_(callback) {}

void UninstallerCore::Report(const std::wstring& status) {
    if (callback_) {
        callback_(status);
    }
}

static std::wstring GetInstallLocationFromRegistry() {
    std::wstring subkey = L"SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\BCTools";
    HKEY hKey = nullptr;
    std::wstring result;

    if (RegOpenKeyExW(HKEY_LOCAL_MACHINE, subkey.c_str(), 0,
                      KEY_QUERY_VALUE | KEY_WOW64_64KEY, &hKey) == ERROR_SUCCESS) {
        wchar_t path[MAX_PATH] = {0};
        DWORD type = 0, size = sizeof(path);
        if (RegQueryValueExW(hKey, L"InstallLocation", nullptr, &type,
                             (LPBYTE)path, &size) == ERROR_SUCCESS && type == REG_SZ) {
            result = path;
        }
        RegCloseKey(hKey);
        if (!result.empty()) return result;
    }

    if (RegOpenKeyExW(HKEY_CURRENT_USER, subkey.c_str(), 0,
                      KEY_QUERY_VALUE, &hKey) == ERROR_SUCCESS) {
        wchar_t path[MAX_PATH] = {0};
        DWORD type = 0, size = sizeof(path);
        if (RegQueryValueExW(hKey, L"InstallLocation", nullptr, &type,
                             (LPBYTE)path, &size) == ERROR_SUCCESS && type == REG_SZ) {
            result = path;
        }
        RegCloseKey(hKey);
    }
    return result;
}

std::wstring UninstallerCore::GetInstallLocationFromIni(const std::wstring& selfPath) {
    std::wstring dir = Utils::GetParentDirectory(selfPath);
    if (dir.empty()) return std::wstring();
    std::wstring iniPath = Utils::JoinPath(dir, L"uninstall.ini");
    wchar_t path[MAX_PATH] = {0};
    DWORD len = GetPrivateProfileStringW(L"Setup", L"InstallPath", L"",
                                         path, MAX_PATH, iniPath.c_str());
    if (len > 0 && path[0] != L'\0') {
        return std::wstring(path);
    }
    return std::wstring();
}

std::wstring UninstallerCore::GetInstallLocation(const std::wstring& selfPath) {
    if (!selfPath.empty()) {
        std::wstring fromIni = GetInstallLocationFromIni(selfPath);
        if (!fromIni.empty()) return fromIni;
    }
    std::wstring fromReg = GetInstallLocationFromRegistry();
    if (!fromReg.empty()) return fromReg;
    if (!selfPath.empty()) {
        return Utils::GetParentDirectory(selfPath);
    }
    return std::wstring();
}

bool UninstallerCore::Run(const std::wstring& selfPath,
                          const std::wstring& installLocationOverride) {
    Report(L"正在查找安装位置…");
    std::wstring installLocation = installLocationOverride;
    if (installLocation.empty()) {
        installLocation = GetInstallLocation(selfPath);
    }
    if (installLocation.empty()) {
        installLocation = Utils::GetParentDirectory(selfPath);
    }

    // 结束运行中的程序
    Report(L"正在停止运行中的程序…");
    Utils::KillProcessByName(L"bctools.exe");
    Utils::KillProcessByName(L"bctool_dev.exe");

    auto pids = Utils::FindPidsByPort(1743);
    for (DWORD pid : pids) {
        Utils::KillProcessByPid(pid);
    }

    // 等待进程释放文件
    Sleep(800);

    // 删除安装目录
    Report(L"正在删除程序文件…");
    if (!installLocation.empty()) {
        DeleteRecursiveInternal(installLocation);
    }

    // 删除数据目录
    Report(L"正在清理本地数据…");
    std::wstring appData = Utils::GetAppData();
    if (!appData.empty()) {
        std::wstring dataDir = Utils::JoinPath(appData, L"BroadcastTool");
        DeleteRecursiveInternal(dataDir);
    }

    // 删除启动项
    Report(L"正在清理启动项…");
    HKEY hRunKey = nullptr;
    if (RegOpenKeyExW(HKEY_CURRENT_USER,
                      L"Software\\Microsoft\\Windows\\CurrentVersion\\Run",
                      0, KEY_SET_VALUE, &hRunKey) == ERROR_SUCCESS) {
        RegDeleteValueW(hRunKey, L"BroadcastTool");
        RegCloseKey(hRunKey);
    }

    // 删除快捷方式
    Report(L"正在删除快捷方式…");
    std::wstring desktop = Utils::GetDesktop();
    if (!desktop.empty()) {
        DeleteFileW(Utils::JoinPath(desktop, L"小播点歌工具.lnk").c_str());
    }
    std::wstring startMenu = Utils::GetStartMenuPrograms();
    if (!startMenu.empty()) {
        std::wstring appMenuDir = Utils::JoinPath(startMenu, L"BCTools");
        DeleteRecursiveInternal(appMenuDir);
    }

    // 删除注册表卸载项
    Report(L"正在清理注册表…");
    Utils::RemoveUninstallRegistry(true);  // HKLM
    Utils::RemoveUninstallRegistry(false); // HKCU

    // 刷新桌面
    Utils::RefreshShell();

    // 安排删除自身（如果是临时副本）
    ScheduleSelfDelete(selfPath);

    Report(L"卸载完成");
    return true;
}
