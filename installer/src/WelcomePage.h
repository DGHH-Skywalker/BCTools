// WelcomePage.h - 欢迎页面
#ifndef WELCOMEPAGE_H
#define WELCOMEPAGE_H

#include "Page.h"
#include "CustomButton.h"
#include <gdiplus.h>
#include <memory>

class WelcomePage : public Page {
public:
    explicit WelcomePage(AppWindow* app);
    ~WelcomePage();

    void Create(HWND parent) override;
    void Layout(const RECT& rect) override;
    void OnEnter() override;

private:
    static LRESULT CALLBACK StaticWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);
    LRESULT HandleMessage(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);

    void Paint(HWND hwnd, HDC hdc);
    void OnInstallClick();

    std::unique_ptr<CustomButton> btnInstall_;
    Gdiplus::Color brandColor_;
};

#endif // WELCOMEPAGE_H
