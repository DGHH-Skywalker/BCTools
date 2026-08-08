// UninstallFinishPage.cpp - 卸载完成页面实现
#include "UninstallFinishPage.h"

#include "AppStrings.h"
#include "AppWindow.h"
#include "Layout.h"
#include "ResourceManager.h"

#include <windowsx.h>

namespace {

constexpr wchar_t kPageClass[] = L"BCToolsUninstallFinishPage";

} // namespace

UninstallFinishPage::UninstallFinishPage(AppWindow* app) : Page(app) {}

void UninstallFinishPage::Create(HWND parent) {
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

    btnFinish_ = std::make_unique<CustomButton>(
        hwnd_, hInst, L"完成", 0, 0, Layout::Scale(140), Layout::Scale(40),
        [this]() { OnFinish(); });
}

void UninstallFinishPage::Layout(const RECT& rect) {
    if (hwnd_) {
        SetWindowPos(hwnd_, nullptr, rect.left, rect.top,
                     rect.right - rect.left, rect.bottom - rect.top,
                     SWP_NOZORDER);
    }

    int w = rect.right - rect.left;
    int h = rect.bottom - rect.top;
    int btnW = Layout::Scale(140);
    int btnH = Layout::Scale(40);
    if (btnFinish_) {
        btnFinish_->Move((w - btnW) / 2, h / 2 + Layout::Scale(80), btnW, btnH);
    }
}

void UninstallFinishPage::OnEnter() {
    // nothing
}

LRESULT CALLBACK UninstallFinishPage::StaticWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    if (msg == WM_CREATE) {
        CREATESTRUCTW* cs = reinterpret_cast<CREATESTRUCTW*>(lParam);
        UninstallFinishPage* page = reinterpret_cast<UninstallFinishPage*>(cs->lpCreateParams);
        SetWindowLongPtrW(hwnd, GWLP_USERDATA, reinterpret_cast<LONG_PTR>(page));
        return 0;
    }
    UninstallFinishPage* page = reinterpret_cast<UninstallFinishPage*>(GetWindowLongPtrW(hwnd, GWLP_USERDATA));
    if (page) {
        return page->HandleMessage(hwnd, msg, wParam, lParam);
    }
    return DefWindowProcW(hwnd, msg, wParam, lParam);
}

LRESULT UninstallFinishPage::HandleMessage(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    switch (msg) {
        case WM_PAINT: {
            PAINTSTRUCT ps;
            HDC hdc = BeginPaint(hwnd, &ps);
            Paint(hwnd, hdc);
            EndPaint(hwnd, &ps);
            return 0;
        }

        case WM_CTLCOLORBTN:
        case WM_CTLCOLORSTATIC: {
            HDC hdc = (HDC)wParam;
            SetBkColor(hdc, RGB(255, 255, 255));
            return (LRESULT)GetStockObject(WHITE_BRUSH);
        }

        default:
            break;
    }
    return DefWindowProcW(hwnd, msg, wParam, lParam);
}

void UninstallFinishPage::Paint(HWND hwnd, HDC hdc) {
    RECT rc;
    GetClientRect(hwnd, &rc);
    int w = rc.right - rc.left;
    int h = rc.bottom - rc.top;

    Gdiplus::Graphics gfx(hdc);
    Gdiplus::SolidBrush white(Gdiplus::Color(255, 255, 255));
    gfx.FillRectangle(&white, 0, 0, w, h);

    std::wstring face = app_->Resources().GetFontFangZheng();
    Gdiplus::Font font(face.c_str(), Layout::ScaleF(28), Gdiplus::FontStyleRegular,
                       Gdiplus::UnitPixel);
    Gdiplus::SolidBrush brush(Gdiplus::Color(50, 50, 50));
    Gdiplus::StringFormat fmt;
    fmt.SetAlignment(Gdiplus::StringAlignmentCenter);
    Gdiplus::RectF titleRc(0, Layout::Scale(180), (Gdiplus::REAL)w, Layout::ScaleF(50));
    gfx.DrawString(L"卸载完成", -1, &font, titleRc, &fmt, &brush);

    Gdiplus::Font subFont(face.c_str(), Layout::ScaleF(17), Gdiplus::FontStyleRegular,
                          Gdiplus::UnitPixel);
    Gdiplus::SolidBrush subBrush(Gdiplus::Color(128, 128, 128));
    Gdiplus::RectF subRc(0, Layout::Scale(240), (Gdiplus::REAL)w, Layout::ScaleF(30));
    std::wstring subText = std::wstring(L"感谢您使用 ") + APP_DISPLAY_NAME;
    gfx.DrawString(subText.c_str(), -1, &subFont, subRc, &fmt, &subBrush);
}

void UninstallFinishPage::OnFinish() {
    PostMessageW(app_->Hwnd(), WM_CLOSE, 0, 0);
}
