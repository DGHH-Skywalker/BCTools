// Utils.h - 通用工具函数
#ifndef UTILS_H
#define UTILS_H

#include <windows.h>
#include <string>
#include <vector>

namespace Utils {

// 宽字符字符串辅助
std::wstring ToWString(const char* str);
std::wstring ToWString(const std::string& str);
std::string ToUtf8(const std::wstring& str);

// 路径拼接（自动处理反斜杠）
std::wstring JoinPath(const std::wstring& a, const std::wstring& b);
std::wstring JoinPath(const std::wstring& a, const std::wstring& b, const std::wstring& c);

// 获取特殊文件夹路径
std::wstring GetLocalAppData();
std::wstring GetAppData();
std::wstring GetStartMenuPrograms();
std::wstring GetDesktop();
std::wstring GetProgramFiles();

// 判断当前进程是否以管理员身份运行
bool IsRunAsAdmin();

// 判断路径是否可写（通过尝试创建临时文件）
bool IsDirectoryWritable(const std::wstring& path);

// 确保目录存在（递归创建）
bool EnsureDirectory(const std::wstring& path);

// 复制资源到文件（从资源内存写入磁盘）
bool WriteResourceToFile(HINSTANCE hInst, int resId, const std::wstring& filePath);

// 删除文件或目录
bool DeleteFileOrEmptyDir(const std::wstring& path);
bool DeleteDirectoryRecursive(const std::wstring& path);

// 创建快捷方式
bool CreateShortcut(const std::wstring& targetPath,
                    const std::wstring& arguments,
                    const std::wstring& shortcutPath,
                    const std::wstring& description,
                    const std::wstring& iconPath,
                    int iconIndex);

// 写入/删除卸载注册表项
bool WriteUninstallRegistry(bool perMachine,
                            const std::wstring& installLocation,
                            const std::wstring& uninstallCmd,
                            const std::wstring& displayIcon,
                            const std::wstring& version,
                            DWORD estimatedSizeKb);
bool RemoveUninstallRegistry(bool perMachine);

// 运行命令并获取输出
std::string RunCommand(const std::wstring& cmd);

// 执行提权重启（带参数）
bool RelaunchElevated(const std::wstring& params);

// 以管理员身份执行外部程序并等待（用于卸载时删除自身）
bool RunElevatedAndWait(const std::wstring& filePath, const std::wstring& params);

// 解析 netstat 输出查找占用端口的 PID
std::vector<DWORD> FindPidsByPort(int port);

// 结束进程
bool KillProcessByName(const std::wstring& name);
bool KillProcessByPid(DWORD pid);

// 刷新桌面/开始菜单
void RefreshShell();

// 获取自身可执行文件路径
std::wstring GetSelfPath();

// 获取父目录
std::wstring GetParentDirectory(const std::wstring& path);

// 获取临时目录下的唯一 .exe 路径
std::wstring GetTempExePath();

// 判断 file 是否位于 dir 目录下（不区分大小写）
bool IsUnderDirectory(const std::wstring& file, const std::wstring& dir);

} // namespace Utils

#endif // UTILS_H
