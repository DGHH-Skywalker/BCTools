// ResourceManager.h - 嵌入式资源管理器
#ifndef RESOURCEMANAGER_H
#define RESOURCEMANAGER_H

#include <windows.h>
#include <gdiplus.h>
#include <vector>
#include <string>

// 字体资源句柄包装
struct EmbeddedFont {
    HANDLE handle = nullptr;
    std::vector<BYTE> data;
    int resId = 0;
    std::wstring name;
};

class ResourceManager {
public:
    ResourceManager();
    ~ResourceManager();

    // 初始化 GDI+ 并加载字体/图片
    bool Initialize(HINSTANCE hInst);
    void Shutdown();

    // 获取 Logo 位图（调用方负责释放）
    Gdiplus::Bitmap* CreateLogoBitmap();

    // 获取 GDI+ 画笔/颜色（可选）
    inline Gdiplus::Color GetBrandColor() const { return brandColor_; }

    // 字体名称（加载后可用于 LOGFONT 的 lfFaceName）
    inline const std::wstring& GetFontJiangXi() const { return fontJiangXiName_; }
    inline const std::wstring& GetFontFangZheng() const { return fontFangZhengName_; }

private:
    bool LoadFont(int resId, EmbeddedFont& font, std::wstring& outName);

    HINSTANCE hInst_ = nullptr;
    ULONG_PTR gdiplusToken_ = 0;
    bool initialized_ = false;

    EmbeddedFont fontJiangXi_;
    EmbeddedFont fontFangZheng_;
    std::wstring fontJiangXiName_;
    std::wstring fontFangZhengName_;

    Gdiplus::Color brandColor_;
};

#endif // RESOURCEMANAGER_H
