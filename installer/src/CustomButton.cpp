// CustomButton.cpp - 统一蓝色圆角按钮实现
#include "CustomButton.h"

#include "Layout.h"

#include <gdiplus.h>
#include <windowsx.h>

namespace {

constexpr wchar_t kBtnClassName[] = L"BCToolsCustomButton";

void AddRoundedRect(Gdiplus::GraphicsPath* path, float x, float y, float w, float h, float r) {
    float d = r * 2.0f;
    path->AddArc(x + w - d, y, d, d, 270.0f, 90.0f);
    path->AddArc(x + w - d, y + h - d, d, d, 0.0f, 90.0f);
    path->AddArc(x, y + h - d, d, d, 90.0f, 90.0f);
    path->AddArc(x, y, d, d, 180.0f, 90.0f);
    path->CloseFigure();
}

} // namespace

void CustomButton::EnsureClassRegistered(HINSTANCE hInst) {
    static bool registered = false;
    if (registered) return;

    WNDCLASSEXW wc = {sizeof(wc)};
    wc.style = CS_HREDRAW | CS_VREDRAW;
    wc.lpfnWndProc = StaticWndProc;
    wc.hInstance = hInst;
    wc.hCursor = LoadCursorW(nullptr, IDC_HAND);
    wc.hbrBackground = (HBRUSH)GetStockObject(WHITE_BRUSH);
    wc.lpszClassName = kBtnClassName;
    RegisterClassExW(&wc);
    registered = true;
}

CustomButton::CustomButton(HWND parent, HINSTANCE hInst, const std::wstring& text,
                           int x, int y, int w, int h,
                           std::function<void()> onClick)
    : text_(text), onClick_(std::move(onClick)) {
    EnsureClassRegistered(hInst);
    hwnd_ = CreateWindowExW(0, kBtnClassName, L"",
                            WS_CHILD | WS_VISIBLE,
                            x, y, w, h,
                            parent, nullptr, hInst, this);
    if (hwnd_) {
        SetWindowPos(hwnd_, HWND_TOP, 0, 0, 0, 0,
                     SWP_NOMOVE | SWP_NOSIZE | SWP_NOACTIVATE);
    }
}

CustomButton::~CustomButton() {
    if (hwnd_) DestroyWindow(hwnd_);
}

void CustomButton::Move(int x, int y, int w, int h) {
    if (hwnd_) {
        SetWindowPos(hwnd_, nullptr, x, y, w, h, SWP_NOZORDER);
    }
}

void CustomButton::SetFontSize(float size) {
    fontSize_ = size;
    if (hwnd_) {
        InvalidateRect(hwnd_, nullptr, FALSE);
    }
}

LRESULT CALLBACK CustomButton::StaticWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    if (msg == WM_CREATE) {
        CREATESTRUCTW* cs = reinterpret_cast<CREATESTRUCTW*>(lParam);
        CustomButton* btn = reinterpret_cast<CustomButton*>(cs->lpCreateParams);
        SetWindowLongPtrW(hwnd, GWLP_USERDATA, reinterpret_cast<LONG_PTR>(btn));
        return 0;
    }
    CustomButton* btn = reinterpret_cast<CustomButton*>(GetWindowLongPtrW(hwnd, GWLP_USERDATA));
    if (btn) {
        return btn->HandleMessage(msg, wParam, lParam);
    }
    return DefWindowProcW(hwnd, msg, wParam, lParam);
}

LRESULT CustomButton::HandleMessage(UINT msg, WPARAM wParam, LPARAM lParam) {
    switch (msg) {
        case WM_PAINT: {
            PAINTSTRUCT ps;
            HDC hdc = BeginPaint(hwnd_, &ps);
            Draw(hdc);
            EndPaint(hwnd_, &ps);
            return 0;
        }

        case WM_MOUSEMOVE:
            SetCursor(LoadCursorW(nullptr, IDC_HAND));
            if (!hover_) {
                hover_ = true;
                InvalidateRect(hwnd_, nullptr, FALSE);
                UpdateTracking();
            }
            return 0;

        case WM_MOUSELEAVE:
            hover_ = false;
            pressed_ = false;
            tracking_ = false;
            InvalidateRect(hwnd_, nullptr, FALSE);
            return 0;

        case WM_LBUTTONDOWN:
            pressed_ = true;
            SetCapture(hwnd_);
            InvalidateRect(hwnd_, nullptr, FALSE);
            return 0;

        case WM_LBUTTONUP:
            if (pressed_) {
                POINT pt;
                GetCursorPos(&pt);
                RECT rc;
                GetWindowRect(hwnd_, &rc);
                if (PtInRect(&rc, pt) && onClick_) {
                    onClick_();
                }
            }
            pressed_ = false;
            ReleaseCapture();
            InvalidateRect(hwnd_, nullptr, FALSE);
            return 0;

        case WM_CAPTURECHANGED:
            if (pressed_) {
                pressed_ = false;
                InvalidateRect(hwnd_, nullptr, FALSE);
            }
            return 0;

        default:
            break;
    }
    return DefWindowProcW(hwnd_, msg, wParam, lParam);
}

void CustomButton::UpdateTracking() {
    TRACKMOUSEEVENT tme = {sizeof(tme)};
    tme.dwFlags = TME_LEAVE;
    tme.hwndTrack = hwnd_;
    TrackMouseEvent(&tme);
    tracking_ = true;
}

void CustomButton::Draw(HDC hdc) {
    RECT rc;
    GetClientRect(hwnd_, &rc);
    int w = rc.right - rc.left;
    int h = rc.bottom - rc.top;

    Gdiplus::Graphics gfx(hdc);
    gfx.SetSmoothingMode(Gdiplus::SmoothingModeAntiAlias);
    gfx.SetTextRenderingHint(Gdiplus::TextRenderingHintAntiAlias);

    Gdiplus::Color brandColor(0, 134, 195);
    Gdiplus::Color base = brandColor;
    if (pressed_) {
        base = Gdiplus::Color(0, 114, 170);
    } else if (hover_) {
        base = Gdiplus::Color(20, 154, 215);
    }
    Gdiplus::SolidBrush brush(base);

    float radius = (float)Layout::Scale(8);
    Gdiplus::GraphicsPath path;
    AddRoundedRect(&path, 0, 0, (float)w, (float)h, radius);
    gfx.FillPath(&brush, &path);

    // 使用系统已安装字体作为按钮文字（方正颜宋或回退到微软雅黑）
    std::wstring face = L"方正颜宋简体";
    Gdiplus::Font font(face.c_str(), Layout::ScaleF(fontSize_),
                       Gdiplus::FontStyleRegular, Gdiplus::UnitPixel);
    Gdiplus::SolidBrush textBrush(Gdiplus::Color(255, 255, 255));
    Gdiplus::StringFormat fmt;
    fmt.SetAlignment(Gdiplus::StringAlignmentCenter);
    fmt.SetLineAlignment(Gdiplus::StringAlignmentCenter);
    Gdiplus::RectF layoutRc(0, 0, (Gdiplus::REAL)w, (Gdiplus::REAL)h);
    gfx.DrawString(text_.c_str(), -1, &font, layoutRc, &fmt, &textBrush);
}
