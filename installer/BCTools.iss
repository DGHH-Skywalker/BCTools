#ifndef MyAppVersion
  #define MyAppVersion "5.8.0.0"
#endif

#define MyAppName "小播点歌工具"
#define MyAppPublisher "Egansama"
#define MyAppExeName "bctools.exe"

[Setup]
; A stable AppId lets Inno Setup detect an existing installation. Running a
; newer, older, or identical package therefore enters upgrade/reinstall mode
; and reuses the previous directory; a missing AppId is a fresh installation.
AppId={{A845EA30-1D47-48D8-A9B1-62C6A4A56CD3}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
VersionInfoVersion={#MyAppVersion}
VersionInfoProductVersion={#MyAppVersion}
DefaultDirName={code:GetDefaultInstallDir}
DefaultGroupName=BCTools
UsePreviousAppDir=yes
UsePreviousGroup=yes
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog commandline
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0
WizardStyle=modern
ShowLanguageDialog=no
Compression=lzma2
SolidCompression=yes
SetupIconFile=..\GoServer\assets\icon-bc.ico
UninstallDisplayIcon={app}\{#MyAppExeName}
UninstallDisplayName={#MyAppName}
OutputDir=..\dist
OutputBaseFilename=BCTools-Setup
CloseApplications=no
RestartApplications=no

[Languages]
Name: "chinesesimplified"; MessagesFile: "languages\ChineseSimplified.isl"

[Messages]
chinesesimplified.PrivilegesRequiredOverrideText1=%1 可以为所有用户安装（需要管理员权限），也可以只为当前用户安装。%n%n其实对101.00%%的该软件用户，这两个选项并没有什么区别
chinesesimplified.PrivilegesRequiredOverrideText2=%1 可以只为当前用户安装，也可以为所有用户安装（需要管理员权限）。%n%n其实对101.00%%的该软件用户，这两个选项并没有什么区别

[Files]
; The Go executable already embeds the Vue/React frontends and ffmpeg tools.
Source: "..\dist\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion

[InstallDelete]
; Clean only legacy application helpers. BctoolData is deliberately excluded.
Type: files; Name: "{app}\bctool_dev.exe"
Type: files; Name: "{app}\uninstall.exe"
Type: files; Name: "{app}\uninstall.ini"
Type: files; Name: "{autoprograms}\BCTools\卸载 {#MyAppName}.lnk"

[Registry]
; The former C++ installer used this key. The stable Inno AppId takes ownership
; after the first successful upgrade, avoiding duplicate uninstall entries.
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Uninstall\BCTools"; Flags: deletekey
Root: HKLM; Subkey: "Software\Microsoft\Windows\CurrentVersion\Uninstall\BCTools"; Flags: deletekey; Check: IsAdminInstallMode

[Icons]
; Stable shortcut paths are overwritten on upgrade/repair, so duplicates are
; never created. Both shortcuts always launch the installed backend executable.
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; WorkingDir: "{app}"
Name: "{autoprograms}\BCTools\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; WorkingDir: "{app}"

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "启动 {#MyAppName}"; Flags: nowait postinstall skipifsilent

[Code]
var
  RemoveUserData: Boolean;

function GetDefaultInstallDir(Param: String): String;
var
  LegacyDir: String;
begin
  // Existing Inno installations are handled by UsePreviousAppDir. These reads
  // migrate installations made by the former C++ installer into the same path.
  if RegQueryStringValue(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\BCTools',
    'InstallLocation', LegacyDir) or
    (IsAdminInstallMode and
      RegQueryStringValue(HKLM, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\BCTools',
        'InstallLocation', LegacyDir)) then
    Result := LegacyDir
  else
    // {autopf} resolves to Program Files for all-users mode and to the current
    // user's Programs directory for non-administrative mode.
    Result := ExpandConstant('{autopf}\BCTools');
end;

function HasSwitch(const Value: String): Boolean;
var
  Index: Integer;
begin
  Result := False;
  for Index := 1 to ParamCount do
    if CompareText(ParamStr(Index), Value) = 0 then
    begin
      Result := True;
      Exit;
    end;
end;

function IsBCToolsRunning: Boolean;
var
  ResultCode: Integer;
  OutputFile: String;
  Output: AnsiString;
begin
  OutputFile := ExpandConstant('{tmp}\bctools-process.txt');
  DeleteFile(OutputFile);
  Exec(ExpandConstant('{cmd}'), '/C tasklist /FI "IMAGENAME eq {#MyAppExeName}" /NH > "' + OutputFile + '" 2>NUL',
    '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Result := LoadStringFromFile(OutputFile, Output) and
    (Pos(Lowercase('{#MyAppExeName}'), Lowercase(String(Output))) > 0);
  DeleteFile(OutputFile);
end;

procedure RequestGracefulShutdown;
var
  ResultCode: Integer;
  Attempt: Integer;
begin
  // BCTools exposes a supported local shutdown endpoint. Ask it to stop first,
  // then wait before allowing Inno Setup to replace any application file.
  Exec('powershell.exe',
    '-NoProfile -NonInteractive -ExecutionPolicy Bypass -Command "try { Invoke-RestMethod -Uri ''http://127.0.0.1:1743/api/shutdown'' -Method Post -TimeoutSec 3 | Out-Null } catch {}"',
    '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  for Attempt := 1 to 20 do
  begin
    if not IsBCToolsRunning then Exit;
    Sleep(250);
  end;
end;

function EnsureBCToolsStopped(IsSilent: Boolean): Boolean;
begin
  Result := True;
  if not IsBCToolsRunning then Exit;

  if not IsSilent then
    if MsgBox('{#MyAppName} 正在运行。安装程序需要先安全关闭它。是否继续？',
      mbConfirmation, MB_OKCANCEL) <> IDOK then
    begin
      Result := False;
      Exit;
    end;

  RequestGracefulShutdown;
  Result := not IsBCToolsRunning;
  if not Result and not IsSilent then
    MsgBox('程序仍在运行。请手动关闭后重新运行安装程序。', mbError, MB_OK);
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
begin
  // This check applies to fresh installs, upgrades, repairs, and silent mode.
  // Silent installation aborts instead of force-killing an unresponsive process.
  if EnsureBCToolsStopped(WizardSilent) then
    Result := ''
  else
    Result := 'BCTools 仍在运行，尚未替换任何文件。';
end;

function InitializeUninstall: Boolean;
begin
  if not EnsureBCToolsStopped(UninstallSilent) then
  begin
    Result := False;
    Exit;
  end;

  // BctoolData is user-owned data beside the executable. Inno Setup removes its
  // own application files and shortcuts but leaves unknown data intact by default.
  // Interactive users may opt in here; silent removal requires /REMOVEUSERDATA.
  RemoveUserData := HasSwitch('/REMOVEUSERDATA');
  if (not UninstallSilent) and (not RemoveUserData) then
    RemoveUserData := MsgBox('是否同时删除 BctoolData 中的歌曲、设置、日志和快照？' + #13#10 +
      '选择“否”会保留全部用户数据。', mbConfirmation, MB_YESNO) = IDYES;
  Result := True;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  // Only an explicit opt-in removes user data. Normal and silent uninstall keep it.
  if (CurUninstallStep = usUninstall) and RemoveUserData then
    DelTree(ExpandConstant('{app}\BctoolData'), True, True, True);
end;
