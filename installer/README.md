# 小播点歌工具安装程序

原生 C++ Win32/GDI+ 安装/卸载程序，无 Web/Electron/.NET 依赖。

## 构建依赖

- MinGW-w64（包含 `g++` 与 `windres`）
- GNU Make

## 资源准备

`build.py` 在构建安装程序前会自动把以下资源复制到 `installer/res/`：

- `GoServer/assets/icon-bc.ico` → `res/icon-bc.ico`
- `Web/public/logo.png` → `res/logo.png`
- `Web/dist/fonts/江西拙楷3.0.ttf` → `res/font_jiangxi_zhuokai.ttf`（构建时按用字精简后的字体）
- `Web/dist/fonts/方正颜宋简体.ttf` → `res/font_fangzheng_yansong.ttf`（构建时按用字精简后的字体）
- `dist/bctools.exe` → `res/bctools.exe`
- `dist/bctool_dev.exe` → `res/bctool_dev.exe`

手动准备时：

```bash
cd installer
cp ../GoServer/assets/icon-bc.ico           res/icon-bc.ico
cp ../Web/public/logo.png                   res/logo.png
cp "../Web/dist/fonts/江西拙楷3.0.ttf"      res/font_jiangxi_zhuokai.ttf
cp "../Web/dist/fonts/方正颜宋简体.ttf"     res/font_fangzheng_yansong.ttf
cp ../dist/bctools.exe                      res/bctools.exe
cp ../dist/bctool_dev.exe                   res/bctool_dev.exe
```

## 编译

```bash
cd installer
make
```

产物为 `installer/setup.exe`。

## 运行

- 安装模式：双击 `setup.exe`
- 卸载模式：`setup.exe /uninstall`

## 页面流程

### 安装模式

1. **WelcomePage**：设计稿首页，左侧 Logo、右侧标题「小播点歌工具」与蓝色圆角安装按钮。
2. **PathPage**：选择安装目录，默认管理员模式 `C:\Program Files\BCTools`，用户模式 `%LOCALAPPDATA%\Programs\BCTools`。
3. **ProgressPage**：释放文件、创建快捷方式、写入卸载注册表。
4. **FinishPage**：完成，可选立即运行。

### 卸载模式

1. **UninstallConfirmPage**：确认卸载并提示删除数据。
2. **UninstallProgressPage**：
   - 结束 `bctools.exe` / `bctool_dev.exe`。
   - 查找并结束占用端口 `1743` 的进程。
   - 删除安装目录、`%APPDATA%\BroadcastTool\`。
   - 删除注册表启动项 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\BroadcastTool`。
   - 删除桌面与开始菜单快捷方式，刷新桌面。
3. **UninstallFinishPage**：完成。

## 权限处理

- 安装程序 manifest 使用 `asInvoker`。
- 若用户选择 `C:\Program Files\BCTools` 且当前非管理员，会尝试以 `runas` 提权重启；用户取消则回退到用户级目录。
- 卸载程序需要管理员权限以删除系统范围注册表项。

## 响应式与高 DPI

- 窗口基准尺寸 960×640。
- 支持 100%/125%/150%/200% DPI 缩放，所有 UI 坐标按 DPI 因子缩放。
- 字体与 Logo 从资源嵌入并运行时加载。

## 已知 TODO

- 卸载时若安装目录无法立即删除（自身占用），使用 `MoveFileEx(MOVEFILE_DELAY_UNTIL_REBOOT)` 延迟删除，需重启生效。
- 字体族名硬编码为文件已知名称，若字体文件变更需同步更新 `ResourceManager.cpp`。
