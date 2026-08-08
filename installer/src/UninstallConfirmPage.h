// UninstallConfirmPage.h - 卸载确认页面
#ifndef UNINSTALLCONFIRMPAGE_H
#define UNINSTALLCONFIRMPAGE_H

#include "Page.h"
#include "CustomButton.h"
#include <windows.h>
#include <memory>

class UninstallConfirmPage : public Page {
public:
    explicit UninstallConfirmPage(AppWindow* app);

    void Create(HWND parent) override;
    void Layout(const RECT& rect) override;
    void OnEnter() override;

private:
    static LRESULT CALLBACK StaticWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);
    LRESULT HandleMessage(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);

    void Paint(HWND hwnd, HDC hdc);
    void OnUninstall();
    void OnCancel();

    std::unique_ptr<CustomButton> btnUninstall_;
    std::unique_ptr<CustomButton> btnCancel_;
};

#endif // UNINSTALLCONFIRMPAGE_H
