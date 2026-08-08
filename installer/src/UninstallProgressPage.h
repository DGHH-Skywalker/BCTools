// UninstallProgressPage.h - 卸载进度页面
#ifndef UNINSTALLPROGRESSPAGE_H
#define UNINSTALLPROGRESSPAGE_H

#include "Page.h"
#include <string>

class UninstallProgressPage : public Page {
public:
    explicit UninstallProgressPage(AppWindow* app);

    void Create(HWND parent) override;
    void Layout(const RECT& rect) override;
    void OnEnter() override;

    HWND Hwnd() const;
    void SetStatus(const std::wstring& status);
    void OnUninstallDone(bool success, const std::wstring& message);

private:
    static LRESULT CALLBACK StaticWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);
    LRESULT HandleMessage(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);

    void Paint(HWND hwnd, HDC hdc);

    HWND hwndStatus_;
    std::wstring status_;
};

#endif // UNINSTALLPROGRESSPAGE_H
