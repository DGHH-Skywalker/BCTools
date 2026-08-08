// ResourceManager.cpp - 嵌入式资源管理器实现
#include "ResourceManager.h"
#include "resource.h"

#include <objidl.h>  // IStream

#pragma comment(lib, "gdiplus.lib")

namespace {

// 从资源创建 GDI+ IStream
IStream* CreateStreamFromResource(HINSTANCE hInst, int resId) {
    HRSRC hRes = FindResourceW(hInst, MAKEINTRESOURCEW(resId), RT_RCDATA);
    if (!hRes) return nullptr;
    HGLOBAL hData = LoadResource(hInst, hRes);
    if (!hData) return nullptr;
    DWORD size = SizeofResource(hInst, hRes);
    void* data = LockResource(hData);
    if (!data || size == 0) return nullptr;

    HGLOBAL hGlobal = GlobalAlloc(GMEM_MOVEABLE, size);
    if (!hGlobal) return nullptr;
    void* ptr = GlobalLock(hGlobal);
    if (!ptr) {
        GlobalFree(hGlobal);
        return nullptr;
    }
    memcpy(ptr, data, size);
    GlobalUnlock(hGlobal);

    IStream* stream = nullptr;
    if (CreateStreamOnHGlobal(hGlobal, TRUE, &stream) != S_OK) {
        GlobalFree(hGlobal);
        return nullptr;
    }
    return stream;
}

} // namespace

ResourceManager::ResourceManager() {
    brandColor_ = Gdiplus::Color(0, 134, 195); // #0086C3
}

ResourceManager::~ResourceManager() {
    Shutdown();
}

bool ResourceManager::Initialize(HINSTANCE hInst) {
    if (initialized_) return true;
    hInst_ = hInst;

    // 初始化 GDI+
    Gdiplus::GdiplusStartupInput input;
    Gdiplus::GdiplusStartup(&gdiplusToken_, &input, nullptr);

    // 加载字体
    if (!LoadFont(IDR_FONT_JIANGXI, fontJiangXi_, fontJiangXiName_)) {
        fontJiangXiName_ = L"江西拙楷3.0"; // 失败时仍尝试使用已知名称
    }
    if (!LoadFont(IDR_FONT_FANGZHENG, fontFangZheng_, fontFangZhengName_)) {
        fontFangZhengName_ = L"方正颜宋简体";
    }

    initialized_ = true;
    return true;
}

void ResourceManager::Shutdown() {
    if (fontJiangXi_.handle) {
        RemoveFontMemResourceEx(fontJiangXi_.handle);
        fontJiangXi_.handle = nullptr;
    }
    if (fontFangZheng_.handle) {
        RemoveFontMemResourceEx(fontFangZheng_.handle);
        fontFangZheng_.handle = nullptr;
    }
    if (gdiplusToken_) {
        Gdiplus::GdiplusShutdown(gdiplusToken_);
        gdiplusToken_ = 0;
    }
    initialized_ = false;
}

bool ResourceManager::LoadFont(int resId, EmbeddedFont& font, std::wstring& outName) {
    HRSRC hRes = FindResourceW(hInst_, MAKEINTRESOURCEW(resId), RT_RCDATA);
    if (!hRes) return false;
    HGLOBAL hData = LoadResource(hInst_, hRes);
    if (!hData) return false;
    DWORD size = SizeofResource(hInst_, hRes);
    void* data = LockResource(hData);
    if (!data || size == 0) return false;

    font.data.resize(size);
    memcpy(font.data.data(), data, size);
    font.resId = resId;

    DWORD loaded = 0;
    font.handle = AddFontMemResourceEx(font.data.data(), (DWORD)font.data.size(),
                                        nullptr, &loaded);
    if (!font.handle) return false;

    // 通过枚举获取字体族名（简单实现：取第一个匹配的 TrueType 字体）
    // 实际字体名可由文件元数据获得；这里使用资源 ID 对应的已知名称。
    if (resId == IDR_FONT_JIANGXI) {
        outName = L"江西拙楷3.0";
    } else if (resId == IDR_FONT_FANGZHENG) {
        outName = L"方正颜宋简体";
    }
    return true;
}

Gdiplus::Bitmap* ResourceManager::CreateLogoBitmap() {
    IStream* stream = CreateStreamFromResource(hInst_, IDR_LOGO_PNG);
    if (!stream) return nullptr;
    Gdiplus::Bitmap* bmp = Gdiplus::Bitmap::FromStream(stream);
    stream->Release();
    return bmp;
}
