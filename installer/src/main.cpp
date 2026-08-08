// main.cpp - 安装程序入口
#include "AppWindow.h"
#include "Utils.h"

#include <windows.h>
#include <shellapi.h>
#include <string>
#include <vector>

// 解析命令行参数
static std::vector<std::wstring> ParseCommandLine() {
    std::vector<std::wstring> args;
    int argc = 0;
    LPWSTR* argv = CommandLineToArgvW(GetCommandLineW(), &argc);
    if (argv) {
        for (int i = 0; i < argc; ++i) {
            args.emplace_back(argv[i]);
        }
        LocalFree(argv);
    }
    return args;
}

// 判断是否为卸载模式
static bool IsUninstallMode(const std::vector<std::wstring>& args) {
    for (const auto& arg : args) {
        if (arg == L"/uninstall" || arg == L"-uninstall" ||
            arg == L"/UNINSTALL" || arg == L"-UNINSTALL") {
            return true;
        }
    }
    return false;
}

// 从命令行获取 /path 参数（安装模式）
static std::wstring GetPathArg(const std::vector<std::wstring>& args) {
    for (size_t i = 0; i + 1 < args.size(); ++i) {
        if (args[i] == L"/path" || args[i] == L"-path") {
            return args[i + 1];
        }
    }
    return std::wstring();
}

// 从命令行获取 /installpath 参数（卸载模式使用的目标路径）
static std::wstring GetInstallPathArg(const std::vector<std::wstring>& args) {
    for (size_t i = 0; i + 1 < args.size(); ++i) {
        if (args[i] == L"/installpath" || args[i] == L"-installpath" ||
            args[i] == L"/INSTALLPATH" || args[i] == L"-INSTALLPATH") {
            return args[i + 1];
        }
    }
    return std::wstring();
}

// 判断是否携带 /skipconfirm（卸载模式跳过确认页）
static bool HasSkipConfirmArg(const std::vector<std::wstring>& args) {
    for (const auto& arg : args) {
        if (arg == L"/skipconfirm" || arg == L"-skipconfirm" ||
            arg == L"/SKIPCONFIRM" || arg == L"-SKIPCONFIRM") {
            return true;
        }
    }
    return false;
}

// 判断文件名是否为 uninstall.exe（用户双击安装目录下的卸载程序时无 /uninstall 参数）
static bool IsUninstallExecutable() {
    std::wstring selfPath = Utils::GetSelfPath();
    size_t pos = selfPath.find_last_of(L"\\/");
    std::wstring name = (pos == std::wstring::npos) ? selfPath : selfPath.substr(pos + 1);
    return _wcsicmp(name.c_str(), L"uninstall.exe") == 0;
}

int WINAPI WinMain(HINSTANCE hInstance, HINSTANCE, LPSTR, int nCmdShow) {
    // 设置每显示器 DPI 感知 V2，避免系统对窗口做位图缩放
    SetThreadDpiAwarenessContext(DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2);

    // 初始化 COM（创建快捷方式需要）
    CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED);

    auto args = ParseCommandLine();
    AppMode mode = (IsUninstallMode(args) || IsUninstallExecutable())
                       ? AppMode::Uninstall
                       : AppMode::Install;

    AppWindow app(hInstance, mode);

    // 如果命令行带 /path，覆盖默认安装路径并直接跳到路径页
    std::wstring pathOverride = GetPathArg(args);
    if (!pathOverride.empty()) {
        app.SetInstallPath(pathOverride);
        app.SetStartAtPathPage(true);
    }

    // 卸载模式：/installpath 指定目标路径，/skipconfirm 跳过确认页
    if (mode == AppMode::Uninstall) {
        std::wstring installPathOverride = GetInstallPathArg(args);
        if (!installPathOverride.empty()) {
            app.SetInstallPath(installPathOverride);
        }
        if (HasSkipConfirmArg(args)) {
            app.SetSkipUninstallConfirm(true);
        }
    }

    if (!app.Create()) {
        CoUninitialize();
        return 1;
    }

    int ret = app.Run();
    CoUninitialize();
    return ret;
}
