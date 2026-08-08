// InstallerCore.h - 安装核心逻辑
#ifndef INSTALLERCORE_H
#define INSTALLERCORE_H

#include <windows.h>
#include <string>
#include <functional>

class InstallerCore {
public:
    using ProgressCallback = std::function<void(int percent, const std::wstring& status)>;

    InstallerCore(HINSTANCE hInst, const std::wstring& installPath,
                  ProgressCallback callback);

    // 执行安装，返回是否成功
    bool Run();

    const std::wstring& LastError() const { return lastError_; }

private:
    bool ReleaseFiles();
    bool CreateShortcuts();
    bool WriteRegistry();
    void Report(int percent, const std::wstring& status);

    HINSTANCE hInst_;
    std::wstring installPath_;
    ProgressCallback callback_;
    std::wstring lastError_;
};

#endif // INSTALLERCORE_H
