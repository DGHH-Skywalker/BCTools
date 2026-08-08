// ProgressPage.cpp - 安装进度页面实现
#include "ProgressPage.h"

#include "AppWindow.h"
#include "Layout.h"
#include "ResourceManager.h"
#include "InstallerCore.h"

#include <windowsx.h>
#include <process.h>
#include <memory>

namespace {

constexpr wchar_t kPageClass[] = L"BCToolsProgressPage";

constexpr UINT WM_UPDATE_PROGRESS = WM_APP + 1;
constexpr UINT WM_INSTALL_DONE = WM_APP + 2;

struct ThreadParam {
    ProgressPage* page;
    std::wstring installPath;
    HINSTANCE hInst;
};

} // namespace

ProgressPage::ProgressPage(AppWindow* app) : Page(app) {}

void ProgressPage::Create(HWND parent) {
    HINSTANCE hInst = app_->Instance();

    WNDCLASSEXW wc = {sizeof(wc)};
    wc.style = CS_HREDRAW | CS_VREDRAW;
    wc.lpfnWndProc = StaticWndProc;
    wc.hInstance = hInst;
    wc.hCursor = LoadCursorW(nullptr, IDC_ARROW);
    wc.hbrBackground = (HBRUSH)GetStockObject(WHITE_BRUSH);
    wc.lpszClassName = kPageClass;
    RegisterClassExW(&wc);

    hwnd_ = CreateWindowExW(0, kPageClass, L"", WS_CHILD | WS_CLIPSIBLINGS,
                            0, 0, 100, 100, parent, nullptr, hInst, this);

    // 状态文本静态控件
    hwndStatus_ = CreateWindowExW(0, L"STATIC", L"",
                                  WS_CHILD | WS_VISIBLE | SS_CENTER,
                                  0, 0, 0, 0, hwnd_, nullptr, hInst, nullptr);

    std::wstring face = app_->Resources().GetFontFangZheng();
    HFONT hFont = CreateFontW(Layout::Scale(17), 0, 0, 0, FW_NORMAL, FALSE, FALSE, FALSE,
                              DEFAULT_CHARSET, OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS,
                              DEFAULT_QUALITY, DEFAULT_PITCH | FF_SWISS, face.c_str());
    SendMessageW(hwndStatus_, WM_SETFONT, (WPARAM)hFont, TRUE);
}

void ProgressPage::Layout(const RECT& rect) {
    if (hwnd_) {
        SetWindowPos(hwnd_, nullptr, rect.left, rect.top,
                     rect.right - rect.left, rect.bottom - rect.top,
                     SWP_NOZORDER);
    }

    int w = rect.right - rect.left;
    int h = rect.bottom - rect.top;
    int margin = Layout::Scale(120);
    int statusH = Layout::Scale(30);
    SetWindowPos(hwndStatus_, nullptr, margin, h / 2 - Layout::Scale(50),
                 w - margin * 2, statusH, SWP_NOZORDER);
}

void ProgressPage::OnEnter() {
    progress_ = 0;
    status_ = L"准备安装…";
    complete_ = false;
    success_ = false;
    message_.clear();
    InvalidateRect(hwnd_, nullptr, TRUE);

    // 启动后台线程执行安装
    ThreadParam* param = new ThreadParam();
    param->page = this;
    param->installPath = app_->InstallPath();
    param->hInst = app_->Instance();

    HANDLE hThread = (HANDLE)_beginthreadex(nullptr, 0,
        [](void* p) -> unsigned int {
            std::unique_ptr<ThreadParam> param(static_cast<ThreadParam*>(p));
            InstallerCore core(param->hInst, param->installPath,
                [&param](int pct, const std::wstring& status) {
                    PostMessageW(param->page->Hwnd(), WM_UPDATE_PROGRESS,
                                 (WPARAM)pct, (LPARAM)new std::wstring(status));
                });
            bool ok = core.Run();
            PostMessageW(param->page->Hwnd(), WM_INSTALL_DONE,
                         (WPARAM)ok, (LPARAM)new std::wstring(core.LastError()));
            return 0;
        }, param, 0, nullptr);
    if (hThread) CloseHandle(hThread);
}

HWND ProgressPage::Hwnd() const {
    return hwnd_;
}

void ProgressPage::SetProgress(int percent, const std::wstring& status) {
    progress_ = percent;
    status_ = status;
    SetWindowTextW(hwndStatus_, status.c_str());
    InvalidateRect(hwnd_, nullptr, TRUE);
}

void ProgressPage::OnInstallComplete(bool success, const std::wstring& message) {
    complete_ = true;
    success_ = success;
    message_ = message;
    if (success) {
        app_->NavigateTo(PageId::Finish);
    } else {
        MessageBoxW(hwnd_, message.c_str(), L"安装失败", MB_OK | MB_ICONERROR);
        app_->NavigateTo(PageId::Path);
    }
}

LRESULT CALLBACK ProgressPage::StaticWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    if (msg == WM_CREATE) {
        CREATESTRUCTW* cs = reinterpret_cast<CREATESTRUCTW*>(lParam);
        ProgressPage* page = reinterpret_cast<ProgressPage*>(cs->lpCreateParams);
        SetWindowLongPtrW(hwnd, GWLP_USERDATA, reinterpret_cast<LONG_PTR>(page));
        return 0;
    }
    ProgressPage* page = reinterpret_cast<ProgressPage*>(GetWindowLongPtrW(hwnd, GWLP_USERDATA));
    if (page) {
        return page->HandleMessage(hwnd, msg, wParam, lParam);
    }
    return DefWindowProcW(hwnd, msg, wParam, lParam);
}

LRESULT ProgressPage::HandleMessage(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    switch (msg) {
        case WM_PAINT: {
            PAINTSTRUCT ps;
            HDC hdc = BeginPaint(hwnd, &ps);
            Paint(hwnd, hdc);
            EndPaint(hwnd, &ps);
            return 0;
        }

        case WM_CTLCOLORSTATIC: {
            HDC hdc = (HDC)wParam;
            SetBkColor(hdc, RGB(255, 255, 255));
            return (LRESULT)GetStockObject(WHITE_BRUSH);
        }

        case WM_UPDATE_PROGRESS: {
            int pct = (int)wParam;
            std::wstring* status = reinterpret_cast<std::wstring*>(lParam);
            SetProgress(pct, *status);
            delete status;
            return 0;
        }

        case WM_INSTALL_DONE: {
            bool ok = (wParam != 0);
            std::wstring* msg = reinterpret_cast<std::wstring*>(lParam);
            OnInstallComplete(ok, *msg);
            delete msg;
            return 0;
        }

        default:
            break;
    }
    return DefWindowProcW(hwnd, msg, wParam, lParam);
}

void ProgressPage::Paint(HWND hwnd, HDC hdc) {
    RECT rc;
    GetClientRect(hwnd, &rc);
    int w = rc.right - rc.left;
    int h = rc.bottom - rc.top;

    Gdiplus::Graphics gfx(hdc);
    Gdiplus::SolidBrush white(Gdiplus::Color(255, 255, 255));
    gfx.FillRectangle(&white, 0, 0, w, h);

    // 标题
    std::wstring face = app_->Resources().GetFontFangZheng();
    Gdiplus::Font font(face.c_str(), Layout::ScaleF(24), Gdiplus::FontStyleRegular,
                       Gdiplus::UnitPixel);
    Gdiplus::SolidBrush brush(Gdiplus::Color(50, 50, 50));
    Gdiplus::StringFormat fmt;
    fmt.SetAlignment(Gdiplus::StringAlignmentCenter);
    Gdiplus::RectF titleRc(0, Layout::Scale(60), (Gdiplus::REAL)w, Layout::ScaleF(40));
    gfx.DrawString(L"正在安装", -1, &font, titleRc, &fmt, &brush);

    DrawProgressBar(hdc);
}

void ProgressPage::DrawProgressBar(HDC hdc) {
    RECT rc;
    GetClientRect(hwnd_, &rc);
    int w = rc.right - rc.left;
    int h = rc.bottom - rc.top;

    int margin = Layout::Scale(60);
    int barW = Layout::Scale(560);
    if (barW > w - margin * 2) {
        barW = w - margin * 2;
    }
    int barH = Layout::Scale(16);
    int x = (w - barW) / 2;
    int y = h / 2;

    Gdiplus::Graphics gfx(hdc);
    gfx.SetSmoothingMode(Gdiplus::SmoothingModeAntiAlias);

    // 背景槽
    Gdiplus::SolidBrush bgBrush(Gdiplus::Color(230, 230, 230));
    int radius = barH / 2;
    Gdiplus::GraphicsPath bgPath;
    bgPath.AddArc(x + barW - radius * 2, y, radius * 2, radius * 2, 270, 90);
    bgPath.AddArc(x + barW - radius * 2, y + barH - radius * 2, radius * 2, radius * 2, 0, 90);
    bgPath.AddArc(x, y + barH - radius * 2, radius * 2, radius * 2, 90, 90);
    bgPath.AddArc(x, y, radius * 2, radius * 2, 180, 90);
    bgPath.CloseFigure();
    gfx.FillPath(&bgBrush, &bgPath);

    // 进度填充
    if (progress_ > 0) {
        int fillW = (int)(barW * (progress_ / 100.0f));
        if (fillW < 2 * radius) fillW = 2 * radius;
        Gdiplus::SolidBrush fillBrush(Gdiplus::Color(0, 134, 195));
        Gdiplus::GraphicsPath fillPath;
        fillPath.AddArc(x + fillW - radius * 2, y, radius * 2, radius * 2, 270, 90);
        fillPath.AddArc(x + fillW - radius * 2, y + barH - radius * 2, radius * 2, radius * 2, 0, 90);
        fillPath.AddArc(x, y + barH - radius * 2, radius * 2, radius * 2, 90, 90);
        fillPath.AddArc(x, y, radius * 2, radius * 2, 180, 90);
        fillPath.CloseFigure();
        gfx.FillPath(&fillBrush, &fillPath);
    }
}
