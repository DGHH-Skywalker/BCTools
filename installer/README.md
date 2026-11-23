# Windows 安装程序

当前安装入口是 `BCTools.iss`，使用 Inno Setup 6。`src/` 和旧 Makefile 中的 C++/Win32 安装器仅保留作历史对照，不参与根构建、CI 或 Release。

## 安装位置与数据

| 安装方式 | 默认程序位置 |
| --- | --- |
| 当前用户 | `%LOCALAPPDATA%\Programs\BCTools\bctools.exe` |
| 所有用户 | `%ProgramFiles%\BCTools\bctools.exe` |

数据目录为 `<安装目录>\BctoolData`。主程序内嵌 Vue、um-react 和 FFmpeg/ffprobe，音频工具在运行时释放到 `BctoolData\bin`。

Inno Setup 只管理程序文件和快捷方式，不把 `BctoolData` 登记为安装文件：

- 升级、重装和普通卸载保留数据；
- 交互卸载可由用户选择是否删除数据；
- 静默卸载只有显式传入 `/REMOVEUSERDATA` 才删除数据。

## 安装、升级与退出

稳定的 Inno `AppId` 用于识别首次安装、升级和重装。再次运行同版本安装包可用于修复安装。桌面和开始菜单快捷方式使用固定名称，升级时原位替换。

替换文件前，安装器会请求现有实例的 `POST http://127.0.0.1:1743/api/shutdown`。进程未退出时，交互安装会要求用户关闭程序，静默安装会中止，不强行覆盖正在使用的文件。

## 构建

安装 Inno Setup 6。`ISCC.exe` 不在常见路径时，将 `ISCC_PATH` 设置为它的完整路径。

```powershell
npm run build
```

主程序已存在时，也可单独构建安装包：

```powershell
npm run build:installer
```

输出为 `dist\BCTools-Setup.exe`。

## 静默命令

当前用户安装：

```powershell
& .\BCTools-Setup.exe /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /CURRENTUSER
```

指定目录：

```powershell
& .\BCTools-Setup.exe /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /CURRENTUSER /DIR="D:\Path With Spaces\BCTools"
```

所有用户安装使用 `/ALLUSERS`，需要相应权限。

卸载并保留数据：

```powershell
& "$env:LOCALAPPDATA\Programs\BCTools\unins000.exe" /VERYSILENT /SUPPRESSMSGBOXES /NORESTART
```

卸载并删除数据：

```powershell
& "$env:LOCALAPPDATA\Programs\BCTools\unins000.exe" /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /REMOVEUSERDATA
```

最后一条会删除用户数据，执行前确认已有备份。
