// FinishPage.h - 安装完成页面
#ifndef FINISHPAGE_H
#define FINISHPAGE_H

#include "Page.h"
#include "CustomButton.h"
#include <windows.h>
#include <memory>

class FinishPage : public Page {
public:
    explicit FinishPage(AppWindow* app);

    void Create(HWND parent) override;
    void Layout(const RECT& rect) override;
    void OnEnter() override;

private:
    static LRESULT CALLBACK StaticWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);
    LRESULT HandleMessage(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);

    void Paint(HWND hwnd, HDC hdc);
    void OnRunNow();
    void OnFinish();

    std::unique_ptr<CustomButton> btnRun_;
    std::unique_ptr<CustomButton> btnFinish_;
};

#endif // FINISHPAGE_H
