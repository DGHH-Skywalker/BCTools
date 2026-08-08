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
    badgeBgColor_ = Gdiplus::Color(230, 245, 252);
    badgeTextColor_ = Gdiplus::Color(0, 134, 195);
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

    // "预览版" 角标（标题右侧）
    std::wstring badgeFace = app_->Resources().GetFontFangZheng();
    Gdiplus::Font badgeFont(badgeFace.c_str(), Layout::ScaleF(16),
                            Gdiplus::FontStyleRegular, Gdiplus::UnitPixel);
    Gdiplus::SolidBrush badgeBrush(badgeTextColor_);
    Gdiplus::SolidBrush badgeBg(badgeBgColor_);
    const wchar_t* badgeText = L"预览版";

    Gdiplus::StringFormat leftFmt;
    leftFmt.SetAlignment(Gdiplus::StringAlignmentNear);
    leftFmt.SetLineAlignment(Gdiplus::StringAlignmentCenter);

    Gdiplus::RectF measureRc(0, 0, 1000, 100);
    gfx.MeasureString(badgeText, -1, &badgeFont, measureRc, &leftFmt, &measureRc);
    float badgeW = measureRc.Width + Layout::ScaleF(16);
    float badgeH = measureRc.Height + Layout::ScaleF(8);

    // 计算标题实际宽高，把角标贴在标题右上角
    Gdiplus::RectF titleMeasureRc(0, 0, 1000, 100);
    gfx.MeasureString(APP_DISPLAY_NAME, -1, &titleFont, titleMeasureRc, &leftFmt, &titleMeasureRc);
    float titleW = titleMeasureRc.Width;
    float titleH = titleMeasureRc.Height;
    float titleX = (w - titleW) / 2.0f;
    float badgeX = titleX + titleW + Layout::ScaleF(12);
    float badgeY = titleRc.Y + (titleRc.Height - titleH) / 2.0f;

    // 若角标超出右边界则放在标题左侧
    if (badgeX + badgeW > w - marginX) {
        badgeX = titleX - badgeW - Layout::ScaleF(12);
    }

    Gdiplus::GraphicsPath badgePath;
    float br = (float)Layout::Scale(4);
    float d = br * 2.0f;
    badgePath.AddArc(badgeX + badgeW - d, badgeY, d, d, 270.0f, 90.0f);
    badgePath.AddArc(badgeX + badgeW - d, badgeY + badgeH - d, d, d, 0.0f, 90.0f);
    badgePath.AddArc(badgeX, badgeY + badgeH - d, d, d, 90.0f, 90.0f);
    badgePath.AddArc(badgeX, badgeY, d, d, 180.0f, 90.0f);
    badgePath.CloseFigure();
    gfx.FillPath(&badgeBg, &badgePath);

    Gdiplus::RectF badgeTextRc(badgeX + Layout::ScaleF(8), badgeY,
                               badgeW - Layout::ScaleF(16), badgeH);
    gfx.DrawString(badgeText, -1, &badgeFont, badgeTextRc, &leftFmt, &badgeBrush);
}

void WelcomePage::OnInstallClick() {
    app_->NavigateTo(PageId::Path);
}
