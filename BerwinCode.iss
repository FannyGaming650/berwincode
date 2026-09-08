; BerwinCode installer (Inno Setup 6). Per-user install, no admin rights.
#define MyAppName "BerwinCode"
#define MyAppVersion "1.6.6"
#define MyAppPublisher "Berwin"
#define MyAppExe "BerwinCode.exe"

[Setup]
AppId={{3B1A2C4D-BerwinCode-1000-000000000001}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={autopf}\{#MyAppName}
PrivilegesRequired=lowest
OutputDir=.
OutputBaseFilename=BerwinCode-Setup-{#MyAppVersion}
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
UninstallDisplayName={#MyAppName}
DisableProgramGroupPage=yes

[Files]
Source: "BerwinCode.exe"; DestDir: "{app}"; Flags: ignoreversion restartreplace
Source: "README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "reference\*"; DestDir: "{code:BerwinRefDir}"; Flags: ignoreversion recursesubdirs


[Icons]
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\{#MyAppExe}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExe}"

[Run]
Filename: "{app}\{#MyAppExe}"; Description: "Launch {#MyAppName}"; Flags: nowait postinstall skipifsilent
Filename: "{app}\{#MyAppExe}"; Parameters: "lock-opencode"; Flags: runhidden; Check: not RemovalSkipped; StatusMsg: "Blocking stock opencode command..."

[UninstallRun]
Filename: "{app}\{#MyAppExe}"; Parameters: "unlock-opencode"; Flags: runhidden skipifdoesntexist

[Code]
const
  MOVEFILE_DELAY_UNTIL_REBOOT = 4;

function MoveFileEx(lpExistingFileName, lpNewFileName: String; dwFlags: DWORD): BOOL;
  external 'MoveFileExW@kernel32.dll stdcall';

procedure RebootDelete(const Path: String);
begin
  MoveFileEx(Path, '', MOVEFILE_DELAY_UNTIL_REBOOT);
end;

procedure ForceDeleteFile(const Path: String);
begin
  if FileExists(Path) then begin
    if not DeleteFile(Path) then
      RebootDelete(Path);
  end;
end;

procedure ForceDeleteDir(const Path: String);
var
  Exe: String;
begin
  if DirExists(Path) then begin
    DelTree(Path, True, True, True);
    Exe := Path + '\bin\opencode.exe';
    if FileExists(Exe) then
      RebootDelete(Exe);
    Exe := Path + '\bin\opencode';
    if FileExists(Exe) then
      RebootDelete(Exe);
  end;
end;

procedure CleanNpmStaging(const NodeModules: String);
var
  FindRec: TFindRec;
begin
  if FindFirst(NodeModules + '\.opencode-ai-*', FindRec) then begin
    try
      repeat
        if (FindRec.Name <> '.') and (FindRec.Name <> '..') then
          DelTree(NodeModules + '\' + FindRec.Name, True, True, True);
      until not FindNext(FindRec);
    finally
      FindClose(FindRec);
    end;
  end;
end;

procedure RemoveStockOpenCode();
var
  NPM, NPMRoot, Home: String;
  ResultCode: Integer;
begin
  Home := GetEnv('USERPROFILE');
  NPMRoot := ExpandConstant('{userappdata}\npm');
  NPM := NPMRoot + '\npm.cmd';
  { 1. proper npm uninstall when available (silent, 90s max) }
  if FileExists(NPM) then begin
    Exec(NPM, 'uninstall -g opencode-ai --silent --no-audit --no-fund', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  end;
  { 2. direct removal of known launchers (never touches node, npm or bun themselves) }
  ForceDeleteFile(NPMRoot + '\opencode.cmd');
  ForceDeleteFile(NPMRoot + '\opencode.ps1');
  ForceDeleteFile(NPMRoot + '\opencode');
  ForceDeleteDir(NPMRoot + '\node_modules\opencode-ai');
  CleanNpmStaging(NPMRoot + '\node_modules');
  ForceDeleteFile(Home + '\.bun\bin\opencode.exe');
  ForceDeleteFile(Home + '\.bun\bin\opencode.cmd');
  ForceDeleteFile(Home + '\.bun\bin\opencode');
  ForceDeleteFile(Home + '\scoop\shims\opencode.exe');
  ForceDeleteFile(Home + '\scoop\shims\opencode.cmd');
  ForceDeleteFile(Home + '\scoop\shims\opencode');
  ForceDeleteDir(Home + '\scoop\apps\opencode');
  ForceDeleteFile(Home + '\.opencode\bin\opencode.exe');
  ForceDeleteFile(Home + '\.opencode\bin\opencode');
  ForceDeleteFile(Home + '\.local\bin\opencode.exe');
  ForceDeleteFile(Home + '\.local\bin\opencode');
end;

function BerwinRefDir(Param: String): String;
begin
  Result := GetEnv('USERPROFILE') + '\.config\berwincode\reference';
end;

function RemovalSkipped(): Boolean;
begin
  Result := (CompareText(ExpandConstant('{param:KEEPOPENCODE|0}'), '1') = 0) or
    (CompareText(GetEnv('BERWINCODE_KEEPOPENCODE'), '1') = 0);
end;

var
  MaintPage: TWizardPage;
  RepairRadio, RemoveRadio, CloseRadio: TRadioButton;

function BerwinUninstallKey(): String;
begin
  Result := 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{3B1A2C4D-BerwinCode-1000-000000000001}_is1';
end;

function IsBerwinInstalled(): Boolean;
begin
  Result := RegKeyExists(HKCU, BerwinUninstallKey());
end;

function InstalledPath(): String;
var
  P: String;
begin
  Result := '';
  if RegQueryStringValue(HKCU, BerwinUninstallKey(), 'Inno Setup: App Path', P) then
    Result := P;
end;

function InstalledVersion(): String;
var
  V: String;
begin
  Result := 'unknown';
  if RegQueryStringValue(HKCU, BerwinUninstallKey(), 'DisplayVersion', V) then
    Result := V;
end;

procedure InitializeWizard();
var
  Info: TNewStaticText;
begin
  MaintPage := CreateCustomPage(wpWelcome, 'BerwinCode is already installed', 'Choose what Setup should do.');
  Info := TNewStaticText.Create(MaintPage);
  Info.Parent := MaintPage.Surface;
  Info.Left := 0;
  Info.Top := 8;
  Info.Width := MaintPage.SurfaceWidth;
  Info.AutoSize := True;
  Info.Caption := 'Setup found BerwinCode ' + InstalledVersion() + ' on this PC.';
  RepairRadio := TRadioButton.Create(MaintPage);
  RepairRadio.Parent := MaintPage.Surface;
  RepairRadio.Left := 0;
  RepairRadio.Top := 40;
  RepairRadio.Width := MaintPage.SurfaceWidth;
  RepairRadio.Caption := '&Repair BerwinCode (reinstall over it)';
  RepairRadio.Checked := True;
  RemoveRadio := TRadioButton.Create(MaintPage);
  RemoveRadio.Parent := MaintPage.Surface;
  RemoveRadio.Left := 0;
  RemoveRadio.Top := 64;
  RemoveRadio.Width := MaintPage.SurfaceWidth;
  RemoveRadio.Caption := 'Re&move BerwinCode from this PC';
  CloseRadio := TRadioButton.Create(MaintPage);
  CloseRadio.Parent := MaintPage.Surface;
  CloseRadio.Left := 0;
  CloseRadio.Top := 88;
  CloseRadio.Width := MaintPage.SurfaceWidth;
  CloseRadio.Caption := '&Close Setup (do nothing)';
end;

function ShouldSkipPage(PageID: Integer): Boolean;
begin
  Result := False;
  if (MaintPage <> nil) and (PageID = MaintPage.ID) and (not IsBerwinInstalled()) then
    Result := True;
end;

function NextButtonClick(CurPageID: Integer): Boolean;
var
  Code: Integer;
  Unins: String;
begin
  Result := True;
  if (MaintPage <> nil) and (CurPageID = MaintPage.ID) then begin
    if RemoveRadio.Checked then begin
      Unins := InstalledPath() + '\unins000.exe';
      if FileExists(Unins) then
        Exec(Unins, '/VERYSILENT /SUPPRESSMSGBOXES /NORESTART', '', SW_HIDE, ewWaitUntilTerminated, Code);
      MsgBox('BerwinCode has been removed from this PC.', mbInformation, MB_OK);
      WizardForm.Close;
      Result := False;
    end else if CloseRadio.Checked then begin
      WizardForm.Close;
      Result := False;
    end else begin
      if InstalledPath() <> '' then
        WizardForm.DirEdit.Text := InstalledPath();
    end;
  end;
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  { Always remove stock opencode unless the owner opted out via flag/env. }
  if (CurStep = ssInstall) and (not RemovalSkipped()) then
    RemoveStockOpenCode();
end;
