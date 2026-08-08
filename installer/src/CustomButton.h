// CustomButton.h - 统一蓝色圆角按钮
#ifndef CUSTOMBUTTON_H
#define CUSTOMBUTTON_H

#include <windows.h>
#include <functional>
#include <string>

class CustomButton {
public:
    CustomButton(HWND parent, HINSTANCE hInst, const std::wstring& text,
                 int x, int y, int w, int h,
                 std::function<void()> onClick);
    ~CustomButton();

    void Move(int x, int y, int w, int h);
    HWND Hwnd() const { return hwnd_; }

    void SetFontSize(float size);

private:
    static void EnsureClassRegistered(HINSTANCE hInst);
    static LRESULT CALLBACK StaticWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);
    LRESULT HandleMessage(UINT msg, WPARAM wParam, LPARAM lParam);

    void Draw(HDC hdc);
    void UpdateTracking();

    HWND hwnd_;
    std::wstring text_;
    std::function<void()> onClick_;
    float fontSize_ = 20.0f;

    bool hover_ = false;
    bool pressed_ = false;
    bool tracking_ = false;
};

#endif // CUSTOMBUTTON_H
