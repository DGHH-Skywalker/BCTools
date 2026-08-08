// Layout.cpp - DPI 缩放与布局计算实现
#include "Layout.h"

int Layout::dpi_ = 96;
float Layout::scaleFactor_ = 1.0f;

void Layout::Init(int dpi) {
    dpi_ = dpi;
    scaleFactor_ = dpi / 96.0f;
}

int Layout::Scale(int value) {
    return (int)(value * scaleFactor_ + 0.5f);
}

float Layout::ScaleF(float value) {
    return value * scaleFactor_;
}

int Layout::Dpi() {
    return dpi_;
}

float Layout::ScaleFactor() {
    return scaleFactor_;
}
