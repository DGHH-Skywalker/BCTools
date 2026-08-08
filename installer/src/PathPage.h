// PathPage.h - 安装路径选择页面
#ifndef PATHPAGE_H
#define PATHPAGE_H

#include "Page.h"
#include "CustomButton.h"
#include <windows.h>
#include <memory>

class PathPage : public Page {
public:
    explicit PathPage(AppWindow* app);

    void Create(HWND parent) override;
    void Layout(const RECT& rect) override;
    void OnEnter() override;

private:
    static LRESULT CALLBACK StaticWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);
    LRESULT HandleMessage(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);

    void Paint(HWND hwnd, HDC hdc);
    void OnBrowse();
    void OnBack();
    void OnInstall();

    HWND hwndEdit_;
    std::unique_ptr<CustomButton> btnBrowse_;
    std::unique_ptr<CustomButton> btnBack_;
    std::unique_ptr<CustomButton> btnInstall_;
};

#endif // PATHPAGE_H
