// UninstallFinishPage.h - 卸载完成页面
#ifndef UNINSTALLFINISHPAGE_H
#define UNINSTALLFINISHPAGE_H

#include "Page.h"
#include "CustomButton.h"
#include <windows.h>
#include <memory>

class UninstallFinishPage : public Page {
public:
    explicit UninstallFinishPage(AppWindow* app);

    void Create(HWND parent) override;
    void Layout(const RECT& rect) override;
    void OnEnter() override;

private:
    static LRESULT CALLBACK StaticWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);
    LRESULT HandleMessage(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);

    void Paint(HWND hwnd, HDC hdc);
    void OnFinish();

    std::unique_ptr<CustomButton> btnFinish_;
};

#endif // UNINSTALLFINISHPAGE_H
