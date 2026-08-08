// UninstallerCore.h - 卸载核心逻辑
#ifndef UNINSTALLERCORE_H
#define UNINSTALLERCORE_H

#include <windows.h>
#include <string>
#include <functional>

class UninstallerCore {
public:
    using StatusCallback = std::function<void(const std::wstring& status)>;

    explicit UninstallerCore(StatusCallback callback);

    // selfPath 为当前 uninstall.exe 路径，installLocationOverride 可强制指定目标路径
    bool Run(const std::wstring& selfPath,
             const std::wstring& installLocationOverride = std::wstring());

    const std::wstring& LastError() const { return lastError_; }

    // 综合获取安装位置：优先 uninstall.ini，其次注册表
    static std::wstring GetInstallLocation(const std::wstring& selfPath = std::wstring());

    // 从 selfPath 同目录下的 uninstall.ini 读取 InstallPath
    static std::wstring GetInstallLocationFromIni(const std::wstring& selfPath);

private:
    void Report(const std::wstring& status);

    StatusCallback callback_;
    std::wstring lastError_;
};

#endif // UNINSTALLERCORE_H
