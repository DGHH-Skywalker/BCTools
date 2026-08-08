// AppWindow.h - 主窗口与页面状态机
#ifndef APPWINDOW_H
#define APPWINDOW_H

#include <windows.h>
#include <string>
#include <memory>
#include <vector>

#include "ResourceManager.h"
#include "CustomButton.h"

class Page;

enum class AppMode {
    Install,
    Uninstall
};

enum class PageId {
    Welcome,
    Path,
    Progress,
    Finish,
    UninstallConfirm,
    UninstallProgress,
    UninstallFinish
};

class AppWindow {
public:
    AppWindow(HINSTANCE hInst, AppMode mode);
    ~AppWindow();

    bool Create();
    int Run();

    // 页面跳转
    void NavigateTo(PageId page);

    // 安装路径
    const std::wstring& InstallPath() const { return installPath_; }
    void SetInstallPath(const std::wstring& path) { installPath_ = path; }

    // 获取句柄与资源管理器
    HWND Hwnd() const { return hwnd_; }
    HINSTANCE Instance() const { return hInst_; }
    ResourceManager& Resources() { return resources_; }
    AppMode Mode() const { return mode_; }

    // 立即运行选项
    bool ShouldRunOnFinish() const { return runOnFinish_; }
    void SetRunOnFinish(bool v) { runOnFinish_ = v; }

    // 若命令行带 /path，安装模式直接从路径页开始
    void SetStartAtPathPage(bool v) { startAtPathPage_ = v; }

    // 卸载模式：/installpath 指定的目标路径；/skipconfirm 直接开始卸载
    void SetSkipUninstallConfirm(bool v) { skipUninstallConfirm_ = v; }

    // 默认安装路径
    static std::wstring GetDefaultInstallPath();

private:
    static LRESULT CALLBACK WndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);
    LRESULT HandleMessage(UINT msg, WPARAM wParam, LPARAM lParam);

    void RegisterWndClass();
    void BuildPages();
    void UpdateWindowSize();
    void UpdateRoundedRegion();
    void LayoutPages();
    void LayoutCloseButton();
    void DrawBackground(HDC hdc);
    int LeftPaneWidth(int totalW) const;

    HINSTANCE hInst_;
    HWND hwnd_;
    AppMode mode_;
    std::wstring title_;
    std::wstring installPath_;
    bool runOnFinish_ = true;
    bool startAtPathPage_ = false;
    bool skipUninstallConfirm_ = false;

    ResourceManager resources_;
    std::unique_ptr<Gdiplus::Bitmap> logo_;
    std::vector<std::unique_ptr<Page>> pages_;
    PageId currentPage_;
    std::unique_ptr<CustomButton> closeBtn_;
};

#endif // APPWINDOW_H
