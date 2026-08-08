// Utils.cpp - 通用工具函数实现
#include "Utils.h"

#include "AppStrings.h"

#include <shlobj.h>
#include <shlwapi.h>
#include <shellapi.h>
#include <objbase.h>
#include <comdef.h>
#include <tlhelp32.h>
#include <stdio.h>
#include <stdlib.h>

#pragma comment(lib, "shell32.lib")
#pragma comment(lib, "shlwapi.lib")
#pragma comment(lib, "advapi32.lib")
#pragma comment(lib, "ole32.lib")
#pragma comment(lib, "uuid.lib")

namespace Utils {

std::wstring ToWString(const char* str) {
    if (!str) return std::wstring();
    int len = MultiByteToWideChar(CP_UTF8, 0, str, -1, nullptr, 0);
    if (len <= 0) return std::wstring();
    std::vector<wchar_t> buf(len);
    MultiByteToWideChar(CP_UTF8, 0, str, -1, buf.data(), len);
    return std::wstring(buf.data());
}

std::wstring ToWString(const std::string& str) {
    return ToWString(str.c_str());
}

std::string ToUtf8(const std::wstring& str) {
    if (str.empty()) return std::string();
    int len = WideCharToMultiByte(CP_UTF8, 0, str.c_str(), (int)str.size(), nullptr, 0, nullptr, nullptr);
    if (len <= 0) return std::string();
    std::string result(len, '\0');
    WideCharToMultiByte(CP_UTF8, 0, str.c_str(), (int)str.size(), &result[0], len, nullptr, nullptr);
    return result;
}

std::wstring JoinPath(const std::wstring& a, const std::wstring& b) {
    if (a.empty()) return b;
    if (b.empty()) return a;
    std::wstring result = a;
    if (result.back() != L'\\' && result.back() != L'/') result += L'\\';
    result += b;
    return result;
}

std::wstring JoinPath(const std::wstring& a, const std::wstring& b, const std::wstring& c) {
    return JoinPath(JoinPath(a, b), c);
}

std::wstring GetFolderPath(int csidl) {
    wchar_t path[MAX_PATH] = {0};
    if (SUCCEEDED(SHGetFolderPathW(nullptr, csidl, nullptr, SHGFP_TYPE_CURRENT, path))) {
        return std::wstring(path);
    }
    return std::wstring();
}

std::wstring GetLocalAppData() {
    return GetFolderPath(CSIDL_LOCAL_APPDATA);
}

std::wstring GetAppData() {
    return GetFolderPath(CSIDL_APPDATA);
}

std::wstring GetStartMenuPrograms() {
    return GetFolderPath(CSIDL_PROGRAMS);
}

std::wstring GetDesktop() {
    return GetFolderPath(CSIDL_DESKTOPDIRECTORY);
}

std::wstring GetProgramFiles() {
    wchar_t path[MAX_PATH] = {0};
    ExpandEnvironmentStringsW(L"%ProgramFiles%", path, MAX_PATH);
    return std::wstring(path);
}

bool IsRunAsAdmin() {
    BOOL isAdmin = FALSE;
    PSID administratorsGroup = nullptr;
    SID_IDENTIFIER_AUTHORITY ntAuthority = SECURITY_NT_AUTHORITY;
    if (AllocateAndInitializeSid(&ntAuthority, 2, SECURITY_BUILTIN_DOMAIN_RID,
                                  DOMAIN_ALIAS_RID_ADMINS, 0, 0, 0, 0, 0, 0,
                                  &administratorsGroup)) {
        CheckTokenMembership(nullptr, administratorsGroup, &isAdmin);
        FreeSid(administratorsGroup);
    }
    return isAdmin != FALSE;
}

bool IsDirectoryWritable(const std::wstring& path) {
    if (path.empty()) return false;
    std::wstring testFile = JoinPath(path, L"_wt_test_.tmp");
    HANDLE h = CreateFileW(testFile.c_str(), GENERIC_WRITE, 0, nullptr, CREATE_ALWAYS,
                           FILE_ATTRIBUTE_NORMAL, nullptr);
    if (h == INVALID_HANDLE_VALUE) return false;
    CloseHandle(h);
    DeleteFileW(testFile.c_str());
    return true;
}

bool EnsureDirectory(const std::wstring& path) {
    if (path.empty()) return false;
    DWORD attr = GetFileAttributesW(path.c_str());
    if (attr != INVALID_FILE_ATTRIBUTES && (attr & FILE_ATTRIBUTE_DIRECTORY)) {
        return true;
    }
    // 递归创建父目录
    std::wstring parent = GetParentDirectory(path);
    if (!parent.empty() && parent != path) {
        if (!EnsureDirectory(parent)) return false;
    }
    return CreateDirectoryW(path.c_str(), nullptr) != 0 || GetLastError() == ERROR_ALREADY_EXISTS;
}

bool WriteResourceToFile(HINSTANCE hInst, int resId, const std::wstring& filePath) {
    HRSRC hRes = FindResourceW(hInst, MAKEINTRESOURCEW(resId), RT_RCDATA);
    if (!hRes) return false;
    HGLOBAL hData = LoadResource(hInst, hRes);
    if (!hData) return false;
    DWORD size = SizeofResource(hInst, hRes);
    void* data = LockResource(hData);
    if (!data || size == 0) return false;

    HANDLE hFile = CreateFileW(filePath.c_str(), GENERIC_WRITE, 0, nullptr, CREATE_ALWAYS,
                               FILE_ATTRIBUTE_NORMAL, nullptr);
    if (hFile == INVALID_HANDLE_VALUE) return false;
    DWORD written = 0;
    BOOL ok = WriteFile(hFile, data, size, &written, nullptr);
    CloseHandle(hFile);
    return ok && written == size;
}

bool DeleteFileOrEmptyDir(const std::wstring& path) {
    if (path.empty()) return false;
    DWORD attr = GetFileAttributesW(path.c_str());
    if (attr == INVALID_FILE_ATTRIBUTES) return true;
    if (attr & FILE_ATTRIBUTE_DIRECTORY) {
        return RemoveDirectoryW(path.c_str()) != 0;
    }
    return DeleteFileW(path.c_str()) != 0;
}

static bool DeleteRecursiveInternal(const std::wstring& path) {
    DWORD attr = GetFileAttributesW(path.c_str());
    if (attr == INVALID_FILE_ATTRIBUTES) return true;
    if (!(attr & FILE_ATTRIBUTE_DIRECTORY)) {
        return DeleteFileW(path.c_str()) != 0;
    }
    std::wstring search = JoinPath(path, L"*");
    WIN32_FIND_DATAW fd;
    HANDLE hFind = FindFirstFileW(search.c_str(), &fd);
    if (hFind == INVALID_HANDLE_VALUE) {
        return RemoveDirectoryW(path.c_str()) != 0;
    }
    do {
        std::wstring name = fd.cFileName;
        if (name == L"." || name == L"..") continue;
        std::wstring full = JoinPath(path, name);
        if (fd.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY) {
            DeleteRecursiveInternal(full);
        } else {
            DeleteFileW(full.c_str());
        }
    } while (FindNextFileW(hFind, &fd));
    FindClose(hFind);
    return RemoveDirectoryW(path.c_str()) != 0;
}

bool DeleteDirectoryRecursive(const std::wstring& path) {
    if (path.empty()) return false;
    return DeleteRecursiveInternal(path);
}

bool CreateShortcut(const std::wstring& targetPath,
                    const std::wstring& arguments,
                    const std::wstring& shortcutPath,
                    const std::wstring& description,
                    const std::wstring& iconPath,
                    int iconIndex) {
    IShellLinkW* pShellLink = nullptr;
    HRESULT hr = CoCreateInstance(CLSID_ShellLink, nullptr, CLSCTX_INPROC_SERVER,
                                  IID_IShellLinkW, (void**)&pShellLink);
    if (FAILED(hr)) return false;

    pShellLink->SetPath(targetPath.c_str());
    if (!arguments.empty()) pShellLink->SetArguments(arguments.c_str());
    if (!description.empty()) pShellLink->SetDescription(description.c_str());
    if (!iconPath.empty()) pShellLink->SetIconLocation(iconPath.c_str(), iconIndex);

    IPersistFile* pPersist = nullptr;
    hr = pShellLink->QueryInterface(IID_IPersistFile, (void**)&pPersist);
    if (SUCCEEDED(hr)) {
        hr = pPersist->Save(shortcutPath.c_str(), TRUE);
        pPersist->Release();
    }
    pShellLink->Release();
    return SUCCEEDED(hr);
}

static HKEY GetUninstallKey(bool perMachine) {
    return perMachine ? HKEY_LOCAL_MACHINE : HKEY_CURRENT_USER;
}

static std::wstring GetUninstallSubkey() {
    return L"SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\BCTools";
}

bool WriteUninstallRegistry(bool perMachine,
                            const std::wstring& installLocation,
                            const std::wstring& uninstallCmd,
                            const std::wstring& displayIcon,
                            const std::wstring& version,
                            DWORD estimatedSizeKb) {
    HKEY root = GetUninstallKey(perMachine);
    std::wstring subkey = GetUninstallSubkey();
    HKEY hKey = nullptr;
    LONG ret = RegCreateKeyExW(root, subkey.c_str(), 0, nullptr, 0,
                               KEY_WRITE, nullptr, &hKey, nullptr);
    if (ret != ERROR_SUCCESS) return false;

    auto setStr = [&](const wchar_t* name, const std::wstring& value) {
        RegSetValueExW(hKey, name, 0, REG_SZ, (const BYTE*)value.c_str(),
                       (DWORD)((value.size() + 1) * sizeof(wchar_t)));
    };
    auto setDword = [&](const wchar_t* name, DWORD value) {
        RegSetValueExW(hKey, name, 0, REG_DWORD, (const BYTE*)&value, sizeof(value));
    };

    setStr(L"DisplayName", APP_DISPLAY_NAME);
    setStr(L"UninstallString", uninstallCmd);
    setStr(L"InstallLocation", installLocation);
    setStr(L"DisplayIcon", displayIcon);
    setStr(L"Publisher", APP_PUBLISHER);
    setStr(L"DisplayVersion", version);
    // InstallDate: YYYYMMDD（“设置-应用”列表显示安装日期）
    {
        SYSTEMTIME st;
        GetLocalTime(&st);
        wchar_t dateBuf[16] = {0};
        _snwprintf(dateBuf, 16, L"%04d%02d%02d", st.wYear, st.wMonth, st.wDay);
        setStr(L"InstallDate", dateBuf);
    }
    setDword(L"NoModify", 1);
    setDword(L"NoRepair", 1);
    setDword(L"EstimatedSize", estimatedSizeKb);

    RegCloseKey(hKey);
    return true;
}

bool RemoveUninstallRegistry(bool perMachine) {
    HKEY root = GetUninstallKey(perMachine);
    std::wstring subkey = GetUninstallSubkey();
    return RegDeleteTreeW(root, subkey.c_str()) == ERROR_SUCCESS ||
           RegDeleteKeyW(root, subkey.c_str()) == ERROR_SUCCESS;
}

std::string RunCommand(const std::wstring& cmd) {
    FILE* pipe = _wpopen(cmd.c_str(), L"r");
    if (!pipe) return std::string();
    std::string result;
    char buffer[1024];
    while (fgets(buffer, sizeof(buffer), pipe)) {
        result += buffer;
    }
    _pclose(pipe);
    return result;
}

bool RelaunchElevated(const std::wstring& params) {
    std::wstring self = GetSelfPath();
    SHELLEXECUTEINFOW sei = {sizeof(sei)};
    sei.lpVerb = L"runas";
    sei.lpFile = self.c_str();
    sei.lpParameters = params.c_str();
    sei.nShow = SW_NORMAL;
    sei.fMask = SEE_MASK_NOCLOSEPROCESS;
    return ShellExecuteExW(&sei) != FALSE;
}

bool RunElevatedAndWait(const std::wstring& filePath, const std::wstring& params) {
    SHELLEXECUTEINFOW sei = {sizeof(sei)};
    sei.lpVerb = L"runas";
    sei.lpFile = filePath.c_str();
    sei.lpParameters = params.c_str();
    sei.nShow = SW_HIDE;
    sei.fMask = SEE_MASK_NOCLOSEPROCESS;
    if (!ShellExecuteExW(&sei)) return false;
    if (sei.hProcess) {
        WaitForSingleObject(sei.hProcess, INFINITE);
        CloseHandle(sei.hProcess);
    }
    return true;
}

std::vector<DWORD> FindPidsByPort(int port) {
    std::vector<DWORD> pids;
    wchar_t cmd[256];
    _snwprintf(cmd, sizeof(cmd) / sizeof(cmd[0]), L"cmd /C netstat -ano | findstr :%d", port);
    std::string output = RunCommand(cmd);
    if (output.empty()) return pids;

    // netstat 输出格式：... 0.0.0.0:1743 LISTENING 1234
    std::string portStr = std::to_string(port);
    size_t pos = 0;
    while (pos < output.size()) {
        size_t end = output.find('\n', pos);
        if (end == std::string::npos) end = output.size();
        std::string line = output.substr(pos, end - pos);
        // 去除 \r
        if (!line.empty() && line.back() == '\r') line.pop_back();
        // 从行尾取最后一个数字作为 PID
        size_t last = line.find_last_of(" \t");
        if (last != std::string::npos) {
            std::string pidStr;
            for (size_t i = last + 1; i < line.size(); ++i) {
                if (line[i] >= '0' && line[i] <= '9') pidStr += line[i];
            }
            if (!pidStr.empty()) {
                pids.push_back((DWORD)std::stoul(pidStr));
            }
        }
        pos = end + 1;
    }
    return pids;
}

bool KillProcessByName(const std::wstring& name) {
    wchar_t cmd[256];
    _snwprintf(cmd, sizeof(cmd) / sizeof(cmd[0]), L"taskkill /IM \"%s\" /F", name.c_str());
    return _wsystem(cmd) == 0;
}

bool KillProcessByPid(DWORD pid) {
    wchar_t cmd[256];
    _snwprintf(cmd, sizeof(cmd) / sizeof(cmd[0]), L"taskkill /PID %lu /F", pid);
    return _wsystem(cmd) == 0;
}

void RefreshShell() {
    SHChangeNotify(SHCNE_ASSOCCHANGED, SHCNF_IDLIST, nullptr, nullptr);
}

std::wstring GetSelfPath() {
    wchar_t path[MAX_PATH] = {0};
    GetModuleFileNameW(nullptr, path, MAX_PATH);
    return std::wstring(path);
}

std::wstring GetParentDirectory(const std::wstring& path) {
    size_t pos = path.find_last_of(L"\\/");
    if (pos == std::wstring::npos) return std::wstring();
    return path.substr(0, pos);
}

std::wstring GetTempExePath() {
    wchar_t tempDir[MAX_PATH] = {0};
    GetTempPathW(MAX_PATH, tempDir);
    wchar_t tempFile[MAX_PATH] = {0};
    if (!GetTempFileNameW(tempDir, L"BCU", 0, tempFile)) return std::wstring();
    std::wstring path(tempFile);
    path += L".exe";
    return path;
}

bool IsUnderDirectory(const std::wstring& file, const std::wstring& dir) {
    if (dir.empty() || file.empty()) return false;
    if (file.size() < dir.size()) return false;
    return _wcsnicmp(file.c_str(), dir.c_str(), dir.size()) == 0;
}

} // namespace Utils
