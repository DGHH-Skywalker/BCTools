// AppWindow.cpp - 主窗口与页面状态机实现
#include "AppWindow.h"

#include "AppStrings.h"
#include "Page.h"
#include "Layout.h"
#include "Utils.h"
#include "resource.h"
#include "WelcomePage.h"
#include "PathPage.h"
#include "ProgressPage.h"
#include "FinishPage.h"
#include "UninstallConfirmPage.h"
#include "UninstallProgressPage.h"
#include "UninstallFinishPage.h"

#include <gdiplus.h>
#include <windowsx.h>
#include <dwmapi.h>

// Win11 圆角属性（旧 SDK 可能缺定义，手动兜底）
#ifndef DWMWA_WINDOW_CORNER_PREFERENCE
#define DWMWA_WINDOW_CORNER_PREFERENCE 33
#endif
#ifndef DWMWCP_ROUND
#define DWMWCP_ROUND 2
#endif

namespace {

constexpr wchar_t kClassName[] = L"BCToolsInstallerWnd";
constexpr int kTitleBarHeight = 44;

} // namespace

AppWindow::AppWindow(HINSTANCE hInst, AppMode mode)
    : hInst_(hInst), hwnd_(nullptr), mode_(mode) {
    title_ = (mode_ == AppMode::Install)
                 ? std::wstring(APP_DISPLAY_NAME) + L" 安装程序"
                 : std::wstring(APP_DISPLAY_NAME) + L" 卸载程序";
    installPath_ = GetDefaultInstallPath();
}

AppWindow::~AppWindow() {
    // pages_ 与 logo_ 自动析构
}

std::wstring AppWindow::GetDefaultInstallPath() {
    if (Utils::IsRunAsAdmin()) {
        return Utils::JoinPath(Utils::GetProgramFiles(), L"BCTools");
    }
    return Utils::JoinPath(Utils::GetLocalAppData(), L"Programs", L"BCTools");
}

void AppWindow::RegisterWndClass() {
    WNDCLASSEXW wc = {sizeof(wc)};
    wc.style = CS_HREDRAW | CS_VREDRAW | CS_DROPSHADOW;
    wc.lpfnWndProc = WndProc;
    wc.hInstance = hInst_;
    wc.hIcon = LoadIconW(hInst_, MAKEINTRESOURCEW(IDI_APP_ICON));
    wc.hCursor = LoadCursorW(nullptr, IDC_ARROW);
    wc.hbrBackground = nullptr; // 背景完全由 WM_PAINT 双缓冲绘制，避免系统默认擦除闪烁
    wc.lpszClassName = kClassName;
    RegisterClassExW(&wc);
}

bool AppWindow::Create() {
    RegisterWndClass();

    // 先按系统 DPI 创建窗口，创建后再以窗口实际 DPI 调整大小
    Layout::Init((int)GetDpiForSystem());

    int width = Layout::Scale(Layout::BASE_WIDTH);
    int height = Layout::Scale(Layout::BASE_HEIGHT);

    hwnd_ = CreateWindowExW(
        WS_EX_APPWINDOW,
        kClassName,
        title_.c_str(),
        WS_POPUP | WS_CLIPCHILDREN,
        CW_USEDEFAULT, CW_USEDEFAULT,
        width, height,
        nullptr, nullptr, hInst_, this);

    if (!hwnd_) return false;

    // 使用窗口所在显示器的 DPI 重新初始化缩放
    Layout::Init((int)GetDpiForWindow(hwnd_));
    UpdateWindowSize();

    // Win11 系统级圆角（半径由系统决定，约 8px，不大）；Win10 自动忽略保持直角
    int cornerPref = DWMWCP_ROUND;
    DwmSetWindowAttribute(hwnd_, DWMWA_WINDOW_CORNER_PREFERENCE,
                          &cornerPref, sizeof(cornerPref));

    // 为无边框窗口启用系统阴影
    MARGINS margins = {0, 0, 0, 1};
    DwmExtendFrameIntoClientArea(hwnd_, &margins);

    // 设置窗口圆角区域（半径不大）
    UpdateRoundedRegion();

    // 居中显示
    RECT rc;
    GetWindowRect(hwnd_, &rc);
    int screenW = GetSystemMetrics(SM_CXSCREEN);
    int screenH = GetSystemMetrics(SM_CYSCREEN);
    int x = (screenW - (rc.right - rc.left)) / 2;
    int y = (screenH - (rc.bottom - rc.top)) / 2;
    SetWindowPos(hwnd_, nullptr, x, y, 0, 0,
                 SWP_NOZORDER | SWP_NOSIZE);

    resources_.Initialize(hInst_);
    logo_.reset(resources_.CreateLogoBitmap());
    BuildPages();

    // 自定义关闭按钮：标题栏右上角，独立于页面之上
    {
        int closeSize = Layout::Scale(36);
        closeBtn_ = std::make_unique<CustomButton>(
            hwnd_, hInst_, L"×", 0, 0, closeSize, closeSize,
            [this]() { PostMessageW(hwnd_, WM_CLOSE, 0, 0); });
        closeBtn_->SetFontSize(Layout::ScaleF(24));
    }

    PageId startPage = PageId::UninstallConfirm;
    if (mode_ == AppMode::Install) {
        startPage = startAtPathPage_ ? PageId::Path : PageId::Welcome;
    } else if (skipUninstallConfirm_) {
        startPage = PageId::UninstallProgress;
    }
    NavigateTo(startPage);

    ShowWindow(hwnd_, SW_SHOW);
    UpdateWindow(hwnd_);
    return true;
}

void AppWindow::BuildPages() {
    if (mode_ == AppMode::Install) {
        pages_.emplace_back(std::make_unique<WelcomePage>(this));
        pages_.emplace_back(std::make_unique<PathPage>(this));
        pages_.emplace_back(std::make_unique<ProgressPage>(this));
        pages_.emplace_back(std::make_unique<FinishPage>(this));
    } else {
        pages_.emplace_back(std::make_unique<UninstallConfirmPage>(this));
        pages_.emplace_back(std::make_unique<UninstallProgressPage>(this));
        pages_.emplace_back(std::make_unique<UninstallFinishPage>(this));
    }

    for (auto& page : pages_) {
        page->Create(hwnd_);
        page->Hide();
    }
}

int AppWindow::LeftPaneWidth(int totalW) const {
    // 左侧面板占 45%，但给右侧内容留至少 480 逻辑像素的空间
    int minRight = Layout::Scale(480);
    int leftW = (int)(totalW * 0.45);
    if (totalW - leftW < minRight) {
        leftW = totalW - minRight;
    }
    if (leftW < Layout::Scale(200)) {
        leftW = Layout::Scale(200);
    }
    return leftW;
}

void AppWindow::LayoutPages() {
    RECT rc;
    GetClientRect(hwnd_, &rc);
    int leftW = LeftPaneWidth(rc.right);
    int rightW = rc.right - leftW;
    int titleH = Layout::Scale(kTitleBarHeight);
    int contentH = rc.bottom - titleH;

    // 把右侧内容区域在父窗口中的绝对坐标传给页面，由页面 Layout 自行定位
    RECT rightRc = {leftW, titleH, leftW + rightW, titleH + contentH};
    for (auto& page : pages_) {
        if (page->Hwnd()) {
            page->Layout(rightRc);
        }
    }

    LayoutCloseButton();

    InvalidateRect(hwnd_, nullptr, FALSE);
}

void AppWindow::LayoutCloseButton() {
    if (!closeBtn_) return;

    RECT rc;
    GetClientRect(hwnd_, &rc);
    int closeSize = Layout::Scale(36);
    int closeMargin = Layout::Scale(8);
    int closeX = rc.right - closeSize - closeMargin;
    closeBtn_->Move(closeX, 60, closeSize, closeSize);
    // 确保关闭按钮始终在所有页面之上
    SetWindowPos(closeBtn_->Hwnd(), HWND_TOP, 0, 0, 0, 0,
                 SWP_NOMOVE | SWP_NOSIZE | SWP_NOACTIVATE);
}

void AppWindow::NavigateTo(PageId page) {
    for (auto& p : pages_) {
        p->Hide();
    }

    Page* target = nullptr;
    switch (page) {
        case PageId::Welcome:
            target = pages_[0].get();
            break;
        case PageId::Path:
            target = pages_[1].get();
            break;
        case PageId::Progress:
            target = pages_[2].get();
            break;
        case PageId::Finish:
            target = pages_[3].get();
            break;
        case PageId::UninstallConfirm:
            target = pages_[0].get();
            break;
        case PageId::UninstallProgress:
            target = pages_[1].get();
            break;
        case PageId::UninstallFinish:
            target = pages_[2].get();
            break;
    }

    if (target) {
        currentPage_ = page;
        LayoutPages();
        target->OnEnter();
        target->Show();
        // 页面显示后会改变 Z 序，必须把关闭按钮重新置顶
        if (closeBtn_) {
            SetWindowPos(closeBtn_->Hwnd(), HWND_TOP, 0, 0, 0, 0,
                         SWP_NOMOVE | SWP_NOSIZE | SWP_NOACTIVATE);
        }
    }
}

int AppWindow::Run() {
    MSG msg;
    while (GetMessageW(&msg, nullptr, 0, 0)) {
        TranslateMessage(&msg);
        DispatchMessageW(&msg);
    }
    return (int)msg.wParam;
}

void AppWindow::UpdateWindowSize() {
    int width = Layout::Scale(Layout::BASE_WIDTH);
    int height = Layout::Scale(Layout::BASE_HEIGHT);
    // 无边框窗口，客户区大小即窗口大小
    SetWindowPos(hwnd_, nullptr, 0, 0, width, height,
                 SWP_NOMOVE | SWP_NOZORDER);
}

void AppWindow::UpdateRoundedRegion() {
    if (!hwnd_) return;
    RECT rc;
    GetWindowRect(hwnd_, &rc);
    int w = rc.right - rc.left;
    int h = rc.bottom - rc.top;
    int radius = Layout::Scale(8);
    HRGN rgn = CreateRoundRectRgn(0, 0, w + 1, h + 1, radius, radius);
    if (rgn) {
        SetWindowRgn(hwnd_, rgn, TRUE);
    }
}

void AppWindow::DrawBackground(HDC hdc) {
    RECT rc;
    GetClientRect(hwnd_, &rc);
    int w = rc.right - rc.left;
    int h = rc.bottom - rc.top;

    Gdiplus::Graphics gfx(hdc);
    gfx.SetSmoothingMode(Gdiplus::SmoothingModeAntiAlias);
    gfx.SetTextRenderingHint(Gdiplus::TextRenderingHintAntiAlias);
    gfx.SetInterpolationMode(Gdiplus::InterpolationModeHighQualityBicubic);

    // 整个背景白色
    Gdiplus::SolidBrush white(Gdiplus::Color(255, 255, 255));
    gfx.FillRectangle(&white, 0, 0, w, h);

    int leftW = LeftPaneWidth(w);
    int titleH = Layout::Scale(kTitleBarHeight);
    int contentY = titleH;
    int contentH = h - titleH;

    // 左侧 Logo 区域：在内容区垂直居中、水平居中
    if (logo_) {
        int margin = Layout::Scale(40);
        int maxLogoW = leftW - margin * 2;
        int maxLogoH = contentH - margin * 2;
        int logoH = Layout::Scale(240);
        int logoW = (int)(logo_->GetWidth() * ((float)logoH / logo_->GetHeight()));
        if (logoW > maxLogoW) {
            logoW = maxLogoW;
            logoH = (int)(logo_->GetHeight() * ((float)logoW / logo_->GetWidth()));
        }
        if (logoH > maxLogoH) {
            logoH = maxLogoH;
            logoW = (int)(logo_->GetWidth() * ((float)logoH / logo_->GetHeight()));
        }
        int logoX = (leftW - logoW) / 2;
        int logoY = contentY + (contentH - logoH) / 2;
        gfx.DrawImage(logo_.get(), logoX, logoY, logoW, logoH);
    }

    // 左右分隔线（从内容区顶部开始）
    Gdiplus::Pen linePen(Gdiplus::Color(230, 230, 230), 1.0f);
    gfx.DrawLine(&linePen, (Gdiplus::REAL)leftW, (Gdiplus::REAL)contentY,
                 (Gdiplus::REAL)leftW, (Gdiplus::REAL)h);

    // 关闭按钮作为独立子窗口绘制，避免父窗口右侧裁剪区无法显示
}

LRESULT CALLBACK AppWindow::WndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    if (msg == WM_CREATE) {
        CREATESTRUCTW* cs = reinterpret_cast<CREATESTRUCTW*>(lParam);
        AppWindow* app = reinterpret_cast<AppWindow*>(cs->lpCreateParams);
        SetWindowLongPtrW(hwnd, GWLP_USERDATA, reinterpret_cast<LONG_PTR>(app));
        return 0;
    }

    AppWindow* app = reinterpret_cast<AppWindow*>(GetWindowLongPtrW(hwnd, GWLP_USERDATA));
    if (app) {
        return app->HandleMessage(msg, wParam, lParam);
    }
    return DefWindowProcW(hwnd, msg, wParam, lParam);
}

LRESULT AppWindow::HandleMessage(UINT msg, WPARAM wParam, LPARAM lParam) {
    switch (msg) {
        case WM_DESTROY:
            PostQuitMessage(0);
            return 0;

        case WM_ERASEBKGND:
            // 背景在 WM_PAINT 中统一绘制，这里直接标记已处理以抑制闪烁
            return 1;

        case WM_PAINT: {
            PAINTSTRUCT ps;
            HDC hdc = BeginPaint(hwnd_, &ps);

            RECT rc;
            GetClientRect(hwnd_, &rc);
            int w = rc.right - rc.left;
            int h = rc.bottom - rc.top;

            HDC memDC = CreateCompatibleDC(hdc);
            HBITMAP memBmp = CreateCompatibleBitmap(hdc, w, h);
            HBITMAP oldBmp = (HBITMAP)SelectObject(memDC, memBmp);

            DrawBackground(memDC);
            BitBlt(hdc, 0, 0, w, h, memDC, 0, 0, SRCCOPY);

            SelectObject(memDC, oldBmp);
            DeleteObject(memBmp);
            DeleteDC(memDC);

            EndPaint(hwnd_, &ps);
            return 0;
        }

        case WM_SIZE:
            UpdateRoundedRegion();
            LayoutPages();
            return 0;

        case WM_NCHITTEST: {
            POINT pt = {GET_X_LPARAM(lParam), GET_Y_LPARAM(lParam)};
            ScreenToClient(hwnd_, &pt);
            RECT rc;
            GetClientRect(hwnd_, &rc);
            int titleH = Layout::Scale(kTitleBarHeight);
            int leftW = LeftPaneWidth(rc.right);
            // 标题栏和左侧 logo 区域支持拖拽无边框窗口
            if (pt.y < titleH || pt.x < leftW) {
                return HTCAPTION;
            }
            break;
        }

        case WM_DPICHANGED: {
            int dpi = HIWORD(wParam);
            Layout::Init(dpi);
            RECT* const prcNew = reinterpret_cast<RECT*>(lParam);
            SetWindowPos(hwnd_, nullptr,
                         prcNew->left, prcNew->top,
                         prcNew->right - prcNew->left,
                         prcNew->bottom - prcNew->top,
                         SWP_NOZORDER);
            UpdateRoundedRegion();
            LayoutPages();
            return 0;
        }

        default:
            break;
    }
    return DefWindowProcW(hwnd_, msg, wParam, lParam);
}
