// FinishPage.cpp - 安装完成页面实现
#include "FinishPage.h"

#include "AppStrings.h"
#include "AppWindow.h"
#include "Layout.h"
#include "ResourceManager.h"
#include "Utils.h"

#include <windowsx.h>

namespace {

constexpr wchar_t kPageClass[] = L"BCToolsFinishPage";

} // namespace

FinishPage::FinishPage(AppWindow* app) : Page(app) {}

void FinishPage::Create(HWND parent) {
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

    btnRun_ = std::make_unique<CustomButton>(
        hwnd_, hInst, L"立即运行", 0, 0, Layout::Scale(140), Layout::Scale(40),
        [this]() { OnRunNow(); });
    btnFinish_ = std::make_unique<CustomButton>(
        hwnd_, hInst, L"完成", 0, 0, Layout::Scale(140), Layout::Scale(40),
        [this]() { OnFinish(); });
}

void FinishPage::Layout(const RECT& rect) {
    if (hwnd_) {
        SetWindowPos(hwnd_, nullptr, rect.left, rect.top,
                     rect.right - rect.left, rect.bottom - rect.top,
                     SWP_NOZORDER);
    }

    int w = rect.right - rect.left;
    int h = rect.bottom - rect.top;
    int btnW = Layout::Scale(140);
    int btnH = Layout::Scale(40);
    int y = h / 2 + Layout::Scale(80);
    int spacing = Layout::Scale(20);
    int totalW = btnW * 2 + spacing;
    int startX = (w - totalW) / 2;

    if (btnRun_) btnRun_->Move(startX, y, btnW, btnH);
    if (btnFinish_) btnFinish_->Move(startX + btnW + spacing, y, btnW, btnH);
}

void FinishPage::OnEnter() {
    app_->SetRunOnFinish(true);
}

LRESULT CALLBACK FinishPage::StaticWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    if (msg == WM_CREATE) {
        CREATESTRUCTW* cs = reinterpret_cast<CREATESTRUCTW*>(lParam);
        FinishPage* page = reinterpret_cast<FinishPage*>(cs->lpCreateParams);
        SetWindowLongPtrW(hwnd, GWLP_USERDATA, reinterpret_cast<LONG_PTR>(page));
        return 0;
    }
    FinishPage* page = reinterpret_cast<FinishPage*>(GetWindowLongPtrW(hwnd, GWLP_USERDATA));
    if (page) {
        return page->HandleMessage(hwnd, msg, wParam, lParam);
    }
    return DefWindowProcW(hwnd, msg, wParam, lParam);
}

LRESULT FinishPage::HandleMessage(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
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

void FinishPage::Paint(HWND hwnd, HDC hdc) {
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
    gfx.DrawString(L"安装完成", -1, &font, titleRc, &fmt, &brush);

    Gdiplus::Font subFont(face.c_str(), Layout::ScaleF(17), Gdiplus::FontStyleRegular,
                          Gdiplus::UnitPixel);
    Gdiplus::SolidBrush subBrush(Gdiplus::Color(128, 128, 128));
    Gdiplus::RectF subRc(0, Layout::Scale(240), (Gdiplus::REAL)w, Layout::ScaleF(30));
    std::wstring subText = std::wstring(L"您现在可以立即运行 ") + APP_DISPLAY_NAME;
    gfx.DrawString(subText.c_str(), -1, &subFont, subRc, &fmt, &subBrush);
}

void FinishPage::OnRunNow() {
    std::wstring exe = Utils::JoinPath(app_->InstallPath(), L"bctools.exe");
    ShellExecuteW(nullptr, L"open", exe.c_str(), nullptr, nullptr, SW_SHOW);
    PostMessageW(app_->Hwnd(), WM_CLOSE, 0, 0);
}

void FinishPage::OnFinish() {
    PostMessageW(app_->Hwnd(), WM_CLOSE, 0, 0);
}
