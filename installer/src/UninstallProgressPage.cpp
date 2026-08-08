// UninstallProgressPage.cpp - 卸载进度页面实现
#include "UninstallProgressPage.h"

#include "AppWindow.h"
#include "Layout.h"
#include "ResourceManager.h"
#include "UninstallerCore.h"
#include "Utils.h"

#include <process.h>
#include <memory>

namespace {

constexpr wchar_t kPageClass[] = L"BCToolsUninstallProgressPage";
constexpr UINT WM_UNINSTALL_STATUS = WM_APP + 10;
constexpr UINT WM_UNINSTALL_DONE = WM_APP + 11;

struct ThreadParam {
    UninstallProgressPage* page;
    std::wstring selfPath;
};

} // namespace

UninstallProgressPage::UninstallProgressPage(AppWindow* app) : Page(app) {}

void UninstallProgressPage::Create(HWND parent) {
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

    hwndStatus_ = CreateWindowExW(0, L"STATIC", L"",
                                  WS_CHILD | WS_VISIBLE | SS_CENTER,
                                  0, 0, 0, 0, hwnd_, nullptr, hInst, nullptr);

    std::wstring face = app_->Resources().GetFontFangZheng();
    HFONT hFont = CreateFontW(Layout::Scale(17), 0, 0, 0, FW_NORMAL, FALSE, FALSE, FALSE,
                              DEFAULT_CHARSET, OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS,
                              DEFAULT_QUALITY, DEFAULT_PITCH | FF_SWISS, face.c_str());
    SendMessageW(hwndStatus_, WM_SETFONT, (WPARAM)hFont, TRUE);
}

void UninstallProgressPage::Layout(const RECT& rect) {
    if (hwnd_) {
        SetWindowPos(hwnd_, nullptr, rect.left, rect.top,
                     rect.right - rect.left, rect.bottom - rect.top,
                     SWP_NOZORDER);
    }

    int w = rect.right - rect.left;
    int h = rect.bottom - rect.top;
    int margin = Layout::Scale(120);
    SetWindowPos(hwndStatus_, nullptr, margin, h / 2 - Layout::Scale(20),
                 w - margin * 2, Layout::Scale(30), SWP_NOZORDER);
}

void UninstallProgressPage::OnEnter() {
    status_ = L"正在准备卸载…";
    SetWindowTextW(hwndStatus_, status_.c_str());
    InvalidateRect(hwnd_, nullptr, TRUE);

    ThreadParam* param = new ThreadParam();
    param->page = this;
    param->selfPath = Utils::GetSelfPath();

    HANDLE hThread = (HANDLE)_beginthreadex(nullptr, 0,
        [](void* p) -> unsigned int {
            std::unique_ptr<ThreadParam> param(static_cast<ThreadParam*>(p));
            UninstallerCore core(
                [&param](const std::wstring& status) {
                    PostMessageW(param->page->Hwnd(), WM_UNINSTALL_STATUS,
                                 0, (LPARAM)new std::wstring(status));
                });
            std::wstring target = param->page->app()->InstallPath();
            bool ok = core.Run(param->selfPath, target);
            PostMessageW(param->page->Hwnd(), WM_UNINSTALL_DONE,
                         (WPARAM)ok, (LPARAM)new std::wstring(core.LastError()));
            return 0;
        }, param, 0, nullptr);
    if (hThread) CloseHandle(hThread);
}

HWND UninstallProgressPage::Hwnd() const {
    return hwnd_;
}

void UninstallProgressPage::SetStatus(const std::wstring& status) {
    status_ = status;
    SetWindowTextW(hwndStatus_, status.c_str());
    InvalidateRect(hwnd_, nullptr, TRUE);
}

void UninstallProgressPage::OnUninstallDone(bool success, const std::wstring& message) {
    if (!success) {
        MessageBoxW(hwnd_, message.c_str(), L"卸载失败", MB_OK | MB_ICONERROR);
    }
    app_->NavigateTo(PageId::UninstallFinish);
}

LRESULT CALLBACK UninstallProgressPage::StaticWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    if (msg == WM_CREATE) {
        CREATESTRUCTW* cs = reinterpret_cast<CREATESTRUCTW*>(lParam);
        UninstallProgressPage* page = reinterpret_cast<UninstallProgressPage*>(cs->lpCreateParams);
        SetWindowLongPtrW(hwnd, GWLP_USERDATA, reinterpret_cast<LONG_PTR>(page));
        return 0;
    }
    UninstallProgressPage* page = reinterpret_cast<UninstallProgressPage*>(GetWindowLongPtrW(hwnd, GWLP_USERDATA));
    if (page) {
        return page->HandleMessage(hwnd, msg, wParam, lParam);
    }
    return DefWindowProcW(hwnd, msg, wParam, lParam);
}

LRESULT UninstallProgressPage::HandleMessage(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
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

        case WM_UNINSTALL_STATUS: {
            std::wstring* status = reinterpret_cast<std::wstring*>(lParam);
            SetStatus(*status);
            delete status;
            return 0;
        }

        case WM_UNINSTALL_DONE: {
            bool ok = (wParam != 0);
            std::wstring* msg = reinterpret_cast<std::wstring*>(lParam);
            OnUninstallDone(ok, *msg);
            delete msg;
            return 0;
        }

        default:
            break;
    }
    return DefWindowProcW(hwnd, msg, wParam, lParam);
}

void UninstallProgressPage::Paint(HWND hwnd, HDC hdc) {
    RECT rc;
    GetClientRect(hwnd, &rc);
    int w = rc.right - rc.left;
    int h = rc.bottom - rc.top;

    Gdiplus::Graphics gfx(hdc);
    Gdiplus::SolidBrush white(Gdiplus::Color(255, 255, 255));
    gfx.FillRectangle(&white, 0, 0, w, h);

    std::wstring face = app_->Resources().GetFontFangZheng();
    Gdiplus::Font font(face.c_str(), Layout::ScaleF(24), Gdiplus::FontStyleRegular,
                       Gdiplus::UnitPixel);
    Gdiplus::SolidBrush brush(Gdiplus::Color(50, 50, 50));
    Gdiplus::StringFormat fmt;
    fmt.SetAlignment(Gdiplus::StringAlignmentCenter);
    Gdiplus::RectF titleRc(0, Layout::Scale(120), (Gdiplus::REAL)w, Layout::ScaleF(40));
    gfx.DrawString(L"正在卸载", -1, &font, titleRc, &fmt, &brush);
}
