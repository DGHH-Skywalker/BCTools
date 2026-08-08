// PathPage.cpp - 安装路径选择页面实现
#include "PathPage.h"

#include "AppStrings.h"
#include "AppWindow.h"
#include "Layout.h"
#include "ResourceManager.h"
#include "Utils.h"

#include <shlobj.h>
#include <commctrl.h>
#include <windowsx.h>

namespace {

constexpr wchar_t kPageClass[] = L"BCToolsPathPage";

} // namespace

PathPage::PathPage(AppWindow* app) : Page(app), hwndEdit_(nullptr) {}

void PathPage::Create(HWND parent) {
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

    // 路径编辑框
    hwndEdit_ = CreateWindowExW(WS_EX_CLIENTEDGE, L"EDIT", L"",
                                WS_CHILD | WS_VISIBLE | ES_AUTOHSCROLL,
                                0, 0, 0, 0, hwnd_, nullptr, hInst, nullptr);

    // 统一蓝色圆角按钮
    btnBrowse_ = std::make_unique<CustomButton>(
        hwnd_, hInst, L"浏览", 0, 0, Layout::Scale(90), Layout::Scale(36),
        [this]() { OnBrowse(); });
    btnBack_ = std::make_unique<CustomButton>(
        hwnd_, hInst, L"上一步", 0, 0, Layout::Scale(100), Layout::Scale(40),
        [this]() { OnBack(); });
    btnInstall_ = std::make_unique<CustomButton>(
        hwnd_, hInst, L"安装", 0, 0, Layout::Scale(100), Layout::Scale(40),
        [this]() { OnInstall(); });

    // 编辑框使用系统默认字体即可
    HFONT hFont = CreateFontW(Layout::Scale(17), 0, 0, 0, FW_NORMAL, FALSE, FALSE, FALSE,
                              DEFAULT_CHARSET, OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS,
                              DEFAULT_QUALITY, DEFAULT_PITCH | FF_SWISS, L"Microsoft YaHei");
    SendMessageW(hwndEdit_, WM_SETFONT, (WPARAM)hFont, TRUE);
}

void PathPage::Layout(const RECT& rect) {
    if (hwnd_) {
        SetWindowPos(hwnd_, nullptr, rect.left, rect.top,
                     rect.right - rect.left, rect.bottom - rect.top,
                     SWP_NOZORDER);
    }

    int w = rect.right - rect.left;
    int h = rect.bottom - rect.top;
    int margin = Layout::Scale(60);
    int editH = Layout::Scale(40);
    int btnW = Layout::Scale(100);
    int btnH = Layout::Scale(40);
    int browseW = Layout::Scale(90);
    int browseH = Layout::Scale(36);
    int spacing = Layout::Scale(12);

    // 编辑框占大部分宽度，右侧浏览按钮
    int editW = w - margin * 2 - browseW - spacing;
    int y = h / 2 - Layout::Scale(20);
    SetWindowPos(hwndEdit_, nullptr, margin, y, editW, editH, SWP_NOZORDER);
    if (btnBrowse_) {
        btnBrowse_->Move(margin + editW + spacing, y + (editH - browseH) / 2, browseW, browseH);
    }

    // 底部按钮
    int bottomY = h - Layout::Scale(80);
    if (btnBack_) {
        btnBack_->Move(margin, bottomY, btnW, btnH);
    }
    if (btnInstall_) {
        btnInstall_->Move(w - margin - btnW, bottomY, btnW, btnH);
    }
}

void PathPage::OnEnter() {
    SetWindowTextW(hwndEdit_, app_->InstallPath().c_str());
}

LRESULT CALLBACK PathPage::StaticWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    if (msg == WM_CREATE) {
        CREATESTRUCTW* cs = reinterpret_cast<CREATESTRUCTW*>(lParam);
        PathPage* page = reinterpret_cast<PathPage*>(cs->lpCreateParams);
        SetWindowLongPtrW(hwnd, GWLP_USERDATA, reinterpret_cast<LONG_PTR>(page));
        return 0;
    }
    PathPage* page = reinterpret_cast<PathPage*>(GetWindowLongPtrW(hwnd, GWLP_USERDATA));
    if (page) {
        return page->HandleMessage(hwnd, msg, wParam, lParam);
    }
    return DefWindowProcW(hwnd, msg, wParam, lParam);
}

LRESULT PathPage::HandleMessage(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    switch (msg) {
        case WM_PAINT: {
            PAINTSTRUCT ps;
            HDC hdc = BeginPaint(hwnd, &ps);
            Paint(hwnd, hdc);
            EndPaint(hwnd, &ps);
            return 0;
        }

        case WM_CTLCOLOREDIT:
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

void PathPage::Paint(HWND hwnd, HDC hdc) {
    RECT rc;
    GetClientRect(hwnd, &rc);
    Gdiplus::Graphics gfx(hdc);
    Gdiplus::SolidBrush white(Gdiplus::Color(255, 255, 255));
    gfx.FillRectangle(&white, 0, 0, rc.right - rc.left, rc.bottom - rc.top);

    // 标题
    std::wstring face = app_->Resources().GetFontFangZheng();
    Gdiplus::Font font(face.c_str(), Layout::ScaleF(26), Gdiplus::FontStyleRegular,
                       Gdiplus::UnitPixel);
    Gdiplus::SolidBrush brush(Gdiplus::Color(50, 50, 50));
    Gdiplus::StringFormat fmt;
    fmt.SetAlignment(Gdiplus::StringAlignmentCenter);
    Gdiplus::RectF titleRc(0, Layout::Scale(60), (Gdiplus::REAL)(rc.right - rc.left), Layout::ScaleF(40));
    gfx.DrawString(L"选择安装位置", -1, &font, titleRc, &fmt, &brush);

    // 提示文字
    Gdiplus::Font tipFont(face.c_str(), Layout::ScaleF(16), Gdiplus::FontStyleRegular,
                          Gdiplus::UnitPixel);
    Gdiplus::SolidBrush tipBrush(Gdiplus::Color(128, 128, 128));
    Gdiplus::RectF tipRc(0, Layout::Scale(110), (Gdiplus::REAL)(rc.right - rc.left), Layout::ScaleF(30));
    std::wstring tipText = std::wstring(L"请选择要安装「") + APP_DISPLAY_NAME + L"」的文件夹";
    gfx.DrawString(tipText.c_str(), -1, &tipFont, tipRc, &fmt, &tipBrush);
}

void PathPage::OnBrowse() {
    BROWSEINFOW bi = {0};
    bi.hwndOwner = hwnd_;
    bi.lpszTitle = L"选择安装文件夹";
    bi.ulFlags = BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE;
    LPITEMIDLIST pidl = SHBrowseForFolderW(&bi);
    if (pidl) {
        wchar_t path[MAX_PATH] = {0};
        if (SHGetPathFromIDListW(pidl, path)) {
            std::wstring newPath = Utils::JoinPath(path, L"BCTools");
            SetWindowTextW(hwndEdit_, newPath.c_str());
        }
        CoTaskMemFree(pidl);
    }
}

void PathPage::OnBack() {
    app_->NavigateTo(PageId::Welcome);
}

void PathPage::OnInstall() {
    wchar_t path[MAX_PATH] = {0};
    GetWindowTextW(hwndEdit_, path, MAX_PATH);
    std::wstring installPath(path);

    if (installPath.empty()) {
        MessageBoxW(hwnd_, L"请选择安装路径", L"提示", MB_OK | MB_ICONINFORMATION);
        return;
    }

    app_->SetInstallPath(installPath);

    // 判断是否需要提权
    bool isProgramFiles = (installPath.find(L"Program Files") != std::wstring::npos) ||
                          (installPath.find(L"ProgramFiles") != std::wstring::npos);
    if (isProgramFiles && !Utils::IsRunAsAdmin()) {
        int ret = MessageBoxW(hwnd_,
                              L"安装到 Program Files 需要管理员权限。是否以管理员身份继续？\n"
                              L"选择「否」将安装到当前用户目录。",
                              L"需要管理员权限",
                              MB_YESNO | MB_ICONQUESTION);
        if (ret == IDYES) {
            std::wstring params = L"/path \"" + installPath + L"\"";
            if (Utils::RelaunchElevated(params)) {
                PostQuitMessage(0);
            } else {
                MessageBoxW(hwnd_, L"提权失败", L"错误", MB_OK | MB_ICONERROR);
            }
            return;
        } else {
            installPath = Utils::JoinPath(Utils::GetLocalAppData(), L"Programs", L"BCTools");
            app_->SetInstallPath(installPath);
            SetWindowTextW(hwndEdit_, installPath.c_str());
        }
    }

    app_->NavigateTo(PageId::Progress);
}
