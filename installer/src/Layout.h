// Layout.h - DPI 缩放与布局计算
#ifndef LAYOUT_H
#define LAYOUT_H

#include <windows.h>

class Layout {
public:
    // 初始化 DPI 缩放（传入窗口 DPI）
    static void Init(int dpi);

    // 基准 96 DPI 下的像素按当前 DPI 缩放
    static int Scale(int value);
    static float ScaleF(float value);

    // 窗口基准尺寸
    static constexpr int BASE_WIDTH = 960;
    static constexpr int BASE_HEIGHT = 640;

    static int Dpi();
    static float ScaleFactor();

private:
    static int dpi_;
    static float scaleFactor_;
};

#endif // LAYOUT_H
