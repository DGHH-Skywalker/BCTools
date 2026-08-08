// Page.h - 页面基类
#ifndef PAGE_H
#define PAGE_H

#include <windows.h>

class AppWindow;

class Page {
public:
    Page(AppWindow* app) : app_(app), hwnd_(nullptr) {}
    virtual ~Page() = default;

    // 创建页面控件，父窗口为传入的 HWND
    virtual void Create(HWND parent) = 0;

    // 页面显示/隐藏
    virtual void Show() { if (hwnd_) ShowWindow(hwnd_, SW_SHOW); }
    virtual void Hide() { if (hwnd_) ShowWindow(hwnd_, SW_HIDE); }

    // 页面尺寸变化时重新布局
    virtual void Layout(const RECT& rect) = 0;

    // 页面进入/离开（可用于加载数据或清理）
    virtual void OnEnter() {}
    virtual void OnLeave() {}

    HWND Hwnd() const { return hwnd_; }
    AppWindow* app() const { return app_; }

protected:
    AppWindow* app_;
    HWND hwnd_;
};

#endif // PAGE_H
