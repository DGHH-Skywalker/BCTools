// UninstallConfirmPage.cpp - 卸载确认页面实现
#include "UninstallConfirmPage.h"

#include "AppStrings.h"
#include "AppWindow.h"
#include "Layout.h"
#include "ResourceManager.h"
#include "UninstallerCore.h"
#include "Utils.h"

#include <shellapi.h>
#include <windowsx.h>

namespace {

constexpr wchar_t kPageClass[] = L"BCToolsUninstallConfirmPage";

} // namespace

UninstallConfirmPage::UninstallConfirmPage(AppWindow* app) : Page(app) {}

void UninstallConfirmPage::Create(HWND parent) {
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

    btnUninstall_ = std::make_unique<CustomButton>(
        hwnd_, hInst, L"卸载", 0, 0, Layout::Scale(140), Layout::Scale(40),
        [this]() { OnUninstall(); });
    btnUninstall_->SetFontSize(24.0f);
    btnCancel_ = std::make_unique<CustomButton>(
        hwnd_, hInst, L"取消", 0, 0, Layout::Scale(140), Layout::Scale(40),
        [this]() { OnCancel(); });
}

void UninstallConfirmPage::Layout(const RECT& rect) {
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

    if (btnUninstall_) btnUninstall_->Move(startX, y, btnW, btnH);
    if (btnCancel_) btnCancel_->Move(startX + btnW + spacing, y, btnW, btnH);
}

void UninstallConfirmPage::OnEnter() {
    // nothing
}

LRESULT CALLBACK UninstallConfirmPage::StaticWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    if (msg == WM_CREATE) {
        CREATESTRUCTW* cs = reinterpret_cast<CREATESTRUCTW*>(lParam);
        UninstallConfirmPage* page = reinterpret_cast<UninstallConfirmPage*>(cs->lpCreateParams);
        SetWindowLongPtrW(hwnd, GWLP_USERDATA, reinterpret_cast<LONG_PTR>(page));
        return 0;
    }
    UninstallConfirmPage* page = reinterpret_cast<UninstallConfirmPage*>(GetWindowLongPtrW(hwnd, GWLP_USERDATA));
    if (page) {
        return page->HandleMessage(hwnd, msg, wParam, lParam);
    }
    return DefWindowProcW(hwnd, msg, wParam, lParam);
}

LRESULT UninstallConfirmPage::HandleMessage(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
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

void UninstallConfirmPage::Paint(HWND hwnd, HDC hdc) {
    RECT rc;
    GetClientRect(hwnd, &rc);
    int w = rc.right - rc.left;
    int h = rc.bottom - rc.top;

    Gdiplus::Graphics gfx(hdc);
    Gdiplus::SolidBrush white(Gdiplus::Color(255, 255, 255));
    gfx.FillRectangle(&white, 0, 0, w, h);

    std::wstring face = app_->Resources().GetFontFangZheng();
    Gdiplus::Font font(face.c_str(), Layout::ScaleF(26), Gdiplus::FontStyleRegular,
                       Gdiplus::UnitPixel);
    Gdiplus::SolidBrush brush(Gdiplus::Color(50, 50, 50));
    Gdiplus::StringFormat fmt;
    fmt.SetAlignment(Gdiplus::StringAlignmentCenter);
    Gdiplus::RectF titleRc(0, Layout::Scale(150), (Gdiplus::REAL)w, Layout::ScaleF(50));
    gfx.DrawString(L"确认卸载", -1, &font, titleRc, &fmt, &brush);

    Gdiplus::Font subFont(face.c_str(), Layout::ScaleF(17), Gdiplus::FontStyleRegular,
                          Gdiplus::UnitPixel);
    Gdiplus::SolidBrush subBrush(Gdiplus::Color(128, 128, 128));
    Gdiplus::RectF subRc(0, Layout::Scale(210), (Gdiplus::REAL)w, Layout::ScaleF(60));
    std::wstring subText = std::wstring(L"此操作将删除 ") + APP_DISPLAY_NAME +
                           L" 及其本地数据。\n是否继续？";
    gfx.DrawString(subText.c_str(), -1, &subFont, subRc, &fmt, &subBrush);
}

void UninstallConfirmPage::OnUninstall() {
    std::wstring selfPath = Utils::GetSelfPath();
    std::wstring installLocation = UninstallerCore::GetInstallLocation(selfPath);
    if (installLocation.empty()) {
        installLocation = Utils::GetParentDirectory(selfPath);
    }

    // 若卸载程序自身位于安装目录内，先复制到临时目录并启动临时副本，
    // 带上目标路径并跳过确认页，然后关闭当前进程，避免原 uninstall.exe 被占用。
    if (!installLocation.empty() && Utils::IsUnderDirectory(selfPath, installLocation)) {
        std::wstring tempExe = Utils::GetTempExePath();
        if (!tempExe.empty() && CopyFileW(selfPath.c_str(), tempExe.c_str(), FALSE)) {
            std::wstring params = L"/uninstall /installpath \"" + installLocation +
                                  L"\" /skipconfirm";
            SHELLEXECUTEINFOW sei = {sizeof(sei)};
            sei.lpVerb = L"open";
            sei.lpFile = tempExe.c_str();
            sei.lpParameters = params.c_str();
            sei.nShow = SW_SHOW;
            sei.fMask = SEE_MASK_NOASYNC;
            if (ShellExecuteExW(&sei)) {
                PostMessageW(app_->Hwnd(), WM_CLOSE, 0, 0);
                return;
            }
        }
    }

    app_->SetInstallPath(installLocation);
    app_->NavigateTo(PageId::UninstallProgress);
}

void UninstallConfirmPage::OnCancel() {
    PostMessageW(app_->Hwnd(), WM_CLOSE, 0, 0);
}
