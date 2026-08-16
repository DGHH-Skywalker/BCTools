// WelcomePage.cpp - 欢迎页面实现
#include "WelcomePage.h"

#include "AppStrings.h"
#include "AppWindow.h"
#include "Layout.h"
#include "ResourceManager.h"

#include <windowsx.h>

namespace {

constexpr wchar_t kPageClass[] = L"BCToolsWelcomePage";

} // namespace

WelcomePage::WelcomePage(AppWindow* app) : Page(app) {
    brandColor_ = Gdiplus::Color(0, 134, 195); // #0086C3
}

WelcomePage::~WelcomePage() = default;

void WelcomePage::Create(HWND parent) {
    WNDCLASSEXW wc = {sizeof(wc)};
    wc.style = CS_HREDRAW | CS_VREDRAW;
    wc.lpfnWndProc = StaticWndProc;
    wc.hInstance = app_->Instance();
    wc.hCursor = LoadCursorW(nullptr, IDC_ARROW);
    wc.hbrBackground = (HBRUSH)GetStockObject(WHITE_BRUSH);
    wc.lpszClassName = kPageClass;
    RegisterClassExW(&wc);

    hwnd_ = CreateWindowExW(0, kPageClass, L"", WS_CHILD | WS_CLIPSIBLINGS,
                            0, 0, 100, 100, parent, nullptr,
                            app_->Instance(), this);

    btnInstall_ = std::make_unique<CustomButton>(
        hwnd_, app_->Instance(), L"安装",
        0, 0, Layout::Scale(240), Layout::Scale(56),
        [this]() { OnInstallClick(); });
    btnInstall_->SetFontSize(26.0f);
}

void WelcomePage::Layout(const RECT& rect) {
    if (hwnd_) {
        SetWindowPos(hwnd_, nullptr, rect.left, rect.top,
                     rect.right - rect.left, rect.bottom - rect.top,
                     SWP_NOZORDER);
    }
    if (btnInstall_) {
        int w = rect.right - rect.left;
        int h = rect.bottom - rect.top;
        int btnW = Layout::Scale(240);
        int btnH = Layout::Scale(56);
        int btnX = (w - btnW) / 2;
        int btnY = h / 2 + Layout::Scale(40);
        btnInstall_->Move(btnX, btnY, btnW, btnH);
    }
}

void WelcomePage::OnEnter() {
    // nothing
}

LRESULT CALLBACK WelcomePage::StaticWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    if (msg == WM_CREATE) {
        CREATESTRUCTW* cs = reinterpret_cast<CREATESTRUCTW*>(lParam);
        WelcomePage* page = reinterpret_cast<WelcomePage*>(cs->lpCreateParams);
        SetWindowLongPtrW(hwnd, GWLP_USERDATA, reinterpret_cast<LONG_PTR>(page));
        return 0;
    }
    WelcomePage* page = reinterpret_cast<WelcomePage*>(GetWindowLongPtrW(hwnd, GWLP_USERDATA));
    if (page) {
        return page->HandleMessage(hwnd, msg, wParam, lParam);
    }
    return DefWindowProcW(hwnd, msg, wParam, lParam);
}

LRESULT WelcomePage::HandleMessage(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    if (msg == WM_PAINT) {
        PAINTSTRUCT ps;
        HDC hdc = BeginPaint(hwnd, &ps);
        Paint(hwnd, hdc);
        EndPaint(hwnd, &ps);
        return 0;
    }
    return DefWindowProcW(hwnd, msg, wParam, lParam);
}

void WelcomePage::Paint(HWND hwnd, HDC hdc) {
    RECT rc;
    GetClientRect(hwnd, &rc);
    int w = rc.right - rc.left;
    int h = rc.bottom - rc.top;

    Gdiplus::Graphics gfx(hdc);
    gfx.SetSmoothingMode(Gdiplus::SmoothingModeAntiAlias);
    gfx.SetTextRenderingHint(Gdiplus::TextRenderingHintAntiAlias);

    // 白底
    Gdiplus::SolidBrush whiteBrush(Gdiplus::Color(255, 255, 255));
    gfx.FillRectangle(&whiteBrush, 0, 0, w, h);

    // 标题
    std::wstring titleFace = app_->Resources().GetFontJiangXi();
    Gdiplus::Font titleFont(titleFace.c_str(), Layout::ScaleF(64),
                            Gdiplus::FontStyleRegular, Gdiplus::UnitPixel);
    Gdiplus::SolidBrush titleBrush(brandColor_);
    Gdiplus::StringFormat titleFmt;
    titleFmt.SetAlignment(Gdiplus::StringAlignmentCenter);
    titleFmt.SetLineAlignment(Gdiplus::StringAlignmentCenter);

    int marginX = Layout::Scale(48);
    Gdiplus::RectF titleRc((Gdiplus::REAL)marginX,
                           (Gdiplus::REAL)(h / 2 - Layout::Scale(60)),
                           (Gdiplus::REAL)(w - marginX * 2),
                           (Gdiplus::REAL)Layout::ScaleF(80));
    gfx.DrawString(APP_DISPLAY_NAME, -1, &titleFont, titleRc, &titleFmt, &titleBrush);
}

void WelcomePage::OnInstallClick() {
    app_->NavigateTo(PageId::Path);
}
