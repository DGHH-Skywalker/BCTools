// ProgressPage.h - 安装进度页面
#ifndef PROGRESSPAGE_H
#define PROGRESSPAGE_H

#include "Page.h"
#include <windows.h>
#include <string>

class ProgressPage : public Page {
public:
    explicit ProgressPage(AppWindow* app);

    void Create(HWND parent) override;
    void Layout(const RECT& rect) override;
    void OnEnter() override;

    // 供后台线程回调更新 UI
    void SetProgress(int percent, const std::wstring& status);
    void OnInstallComplete(bool success, const std::wstring& message);

    HWND Hwnd() const;

private:
    static LRESULT CALLBACK StaticWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);
    LRESULT HandleMessage(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);

    void Paint(HWND hwnd, HDC hdc);
    void DrawProgressBar(HDC hdc);

    HWND hwndStatus_;
    int progress_ = 0;
    std::wstring status_;
    bool complete_ = false;
    bool success_ = false;
    std::wstring message_;
};

#endif // PROGRESSPAGE_H
