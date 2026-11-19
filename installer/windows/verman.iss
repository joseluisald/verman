#ifndef ProductVersion
  #define ProductVersion "0.5.3"
#endif
#ifndef PayloadDir
  #error Informe /DPayloadDir com a pasta preparada pelo build-windows.ps1
#endif
#ifndef OutputDir
  #define OutputDir "..\..\dist"
#endif

[Setup]
AppId={{A18A16A0-15A7-41DA-9732-65FD2D06B62A}
AppName=Verman
AppVersion={#ProductVersion}
AppPublisher=Verman
DefaultDirName={code:DefaultInstallDir}
UsePreviousAppDir=no
DefaultGroupName=Verman
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64os
ArchitecturesInstallIn64BitMode=x64os
MinVersion=10.0.17763
WizardStyle=modern
SetupIconFile={#PayloadDir}\assets\verman.ico
WizardSmallImageFile={#PayloadDir}\assets\verman-logo.png
WizardImageFile={#PayloadDir}\assets\wizard-logo.png
Compression=lzma2
SolidCompression=yes
OutputDir={#OutputDir}
OutputBaseFilename=Verman-Setup-{#ProductVersion}-windows-amd64
UninstallDisplayIcon={app}\assets\verman.ico
CloseApplications=yes
RestartApplications=no
ChangesEnvironment=yes
SetupLogging=yes
InfoBeforeFile=welcome.txt
DisableWelcomePage=no

[Languages]
Name: "brazilianportuguese"; MessagesFile: "compiler:Languages\BrazilianPortuguese.isl"

[Tasks]
Name: "path"; Description: "Adicionar {app}\bin ao PATH do usuário (recomendado)"; GroupDescription: "Para usar verman e as ferramentas selecionadas no terminal:"; Flags: checkedonce
Name: "desktopicon"; Description: "Criar atalho na área de trabalho"; GroupDescription: "Atalhos:"; Flags: unchecked

[Files]
Source: "{#PayloadDir}\verman.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#PayloadDir}\verman-desktop.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#PayloadDir}\assets\*"; DestDir: "{app}\assets"; Flags: ignoreversion
Source: "{#PayloadDir}\bin\*"; DestDir: "{app}\bin"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#PayloadDir}\docs\*"; DestDir: "{app}\docs"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#PayloadDir}\scripts\*"; DestDir: "{app}\scripts"; Flags: ignoreversion
Source: "{#PayloadDir}\docs\*.html"; Flags: dontcopy

[Icons]
Name: "{group}\Verman"; Filename: "{app}\verman-desktop.exe"; WorkingDir: "{app}"; IconFilename: "{app}\assets\verman.ico"
Name: "{group}\Primeiros passos"; Filename: "{app}\docs\PRIMEIROS_PASSOS.html"
Name: "{group}\Documentação"; Filename: "{app}\docs\index.html"
Name: "{group}\Desinstalar Verman"; Filename: "{uninstallexe}"
Name: "{autodesktop}\Verman"; Filename: "{app}\verman-desktop.exe"; WorkingDir: "{app}"; IconFilename: "{app}\assets\verman.ico"; Tasks: desktopicon

[Registry]
Root: HKCU; Subkey: "Software\Verman"; ValueType: string; ValueName: "InstallDir"; ValueData: "{app}"; Flags: uninsdeletekeyifempty uninsdeletevalue
Root: HKCU; Subkey: "Software\Verman"; ValueType: string; ValueName: "Version"; ValueData: "{#ProductVersion}"; Flags: uninsdeletevalue

[Run]
Filename: "{app}\verman-desktop.exe"; Description: "Abrir Verman"; Flags: nowait postinstall skipifsilent runasoriginaluser; Check: HasWebView
Filename: "{app}\docs\PRIMEIROS_PASSOS.html"; Description: "Abrir guia de configuração inicial"; Flags: shellexec postinstall skipifsilent runasoriginaluser
Filename: "{app}\docs\index.html"; Description: "Abrir documentação offline"; Flags: shellexec postinstall skipifsilent runasoriginaluser unchecked
Filename: "https://developer.microsoft.com/microsoft-edge/webview2/"; Description: "Obter Microsoft WebView2 para abrir o desktop"; Flags: shellexec postinstall skipifsilent runasoriginaluser; Check: MissingWebView

[Code]
var GuideButton, DocsButton: TNewButton;
    PrereqLabel: TNewStaticText;
    PathConfigured: Boolean;

function DefaultInstallDir(Param: String): String;
var PreviousDir: String;
begin
  Result := 'C:\verman';
  if RegQueryStringValue(HKCU, 'Software\Verman', 'InstallDir', PreviousDir) and
    FileExists(AddBackslash(PreviousDir) + 'verman.exe') then
    Result := PreviousDir;
end;

function HasWebView: Boolean;
var Version: String;
begin
  Result := (RegQueryStringValue(HKLM32, 'Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}', 'pv', Version) and (Version <> '') and (Version <> '0.0.0.0')) or
    (RegQueryStringValue(HKCU, 'Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}', 'pv', Version) and (Version <> '') and (Version <> '0.0.0.0'));
end;

function MissingWebView: Boolean;
begin
  Result := not HasWebView;
end;

function ManagedPath: Boolean;
var Value: Cardinal;
begin
  Result := RegQueryDWordValue(HKCU, 'Software\Verman', 'PathManaged', Value) and (Value = 1);
end;

procedure ExtractHelp;
begin
  ExtractTemporaryFile('index.html');
  ExtractTemporaryFile('PRIMEIROS_PASSOS.html');
  ExtractTemporaryFile('SUPORTE.html');
  ExtractTemporaryFile('CLI_JSON.html');
  ExtractTemporaryFile('PROVEDORES.html');
  ExtractTemporaryFile('DESKTOP.html');
end;

procedure ShowGuide(Sender: TObject);
var ErrorCode: Integer;
begin
  ExtractHelp;
  ShellExecAsOriginalUser('open', ExpandConstant('{tmp}\PRIMEIROS_PASSOS.html'), '', '', SW_SHOWNORMAL, ewNoWait, ErrorCode);
end;

procedure ShowDocs(Sender: TObject);
var ErrorCode: Integer;
begin
  ExtractHelp;
  ShellExecAsOriginalUser('open', ExpandConstant('{tmp}\index.html'), '', '', SW_SHOWNORMAL, ewNoWait, ErrorCode);
end;

procedure InitializeWizard;
begin
  GuideButton := TNewButton.Create(WizardForm);
  GuideButton.Parent := WizardForm;
  GuideButton.Caption := 'Como começar';
  GuideButton.SetBounds(ScaleX(12), WizardForm.CancelButton.Top, ScaleX(120), WizardForm.CancelButton.Height);
  GuideButton.OnClick := @ShowGuide;
  DocsButton := TNewButton.Create(WizardForm);
  DocsButton.Parent := WizardForm;
  DocsButton.Caption := 'Documentação';
  DocsButton.SetBounds(ScaleX(140), WizardForm.CancelButton.Top, ScaleX(120), WizardForm.CancelButton.Height);
  DocsButton.OnClick := @ShowDocs;
  PrereqLabel := TNewStaticText.Create(WizardForm);
  PrereqLabel.Parent := WizardForm.WelcomePage;
  PrereqLabel.WordWrap := True;
  PrereqLabel.AutoSize := False;
  PrereqLabel.SetBounds(WizardForm.WelcomeLabel2.Left, ScaleY(200), WizardForm.WelcomeLabel2.Width, ScaleY(70));
  if HasWebView then
    PrereqLabel.Caption := 'Microsoft WebView2 encontrado. A CLI e o desktop podem ser instalados.'
  else
    PrereqLabel.Caption := 'Microsoft WebView2 não foi encontrado. A CLI funciona sem ele. Ao concluir, você poderá abrir a página oficial para instalar o runtime exigido pelo desktop.';
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var LegacyDir: String;
begin
  Result := '';
  if RegQueryStringValue(HKLM64, 'Software\Verman', 'InstallDir', LegacyDir) and
    FileExists(AddBackslash(LegacyDir) + 'verman.exe') then begin
    Log('Instalação anterior para todos os usuários em: ' + LegacyDir);
    if CompareText(AddBackslash(ExpandFileName(WizardDirValue)), AddBackslash(ExpandFileName(LegacyDir))) = 0 then begin
      Result := 'O destino já contém uma instalação para todos os usuários. Desinstale a edição anterior em Aplicativos instalados ou escolha outra pasta.';
      Exit;
    end;
    if not WizardSilent then
      MsgBox('Foi encontrada uma instalação anterior para todos os usuários. Ela pode ter prioridade no PATH. Para migrar, desinstale a edição anterior em Aplicativos instalados. Os runtimes e os projetos serão preservados.', mbInformation, MB_OK);
  end;
end;

procedure CurStepChanged(CurStep: TSetupStep);
var ExitCode: Integer;
begin
  if (CurStep = ssPostInstall) and WizardIsTaskSelected('path') then begin
    if Exec(ExpandConstant('{sys}\WindowsPowerShell\v1.0\powershell.exe'),
      ExpandConstant('-NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "{app}\scripts\setup-path.ps1" -Scope User -AllowMachinePathConflicts -VermanExecutable "{app}\verman.exe" -BinDir "{app}\bin"'),
      '', SW_HIDE, ewWaitUntilTerminated, ExitCode) and (ExitCode = 0) then begin
      RegWriteDWordValue(HKCU, 'Software\Verman', 'PathManaged', 1);
      PathConfigured := True;
      Log('PATH do Verman configurado. Outros terminais precisam ser reabertos uma vez.');
    end else begin
      Log('Falha ao configurar PATH. Os aplicativos foram instalados; consulte os primeiros passos.');
      if not WizardSilent then MsgBox('CLI e desktop instalados, mas a configuração do PATH falhou. Abra o guia de primeiros passos para configurar e diagnosticar. Nenhuma instalação externa foi removida.', mbError, MB_OK);
    end;
  end;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var ExitCode: Integer;
begin
  if (CurUninstallStep = usUninstall) and ManagedPath then begin
    if not Exec(ExpandConstant('{sys}\WindowsPowerShell\v1.0\powershell.exe'),
      ExpandConstant('-NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "{app}\scripts\remove-path.ps1" -Scope User -BinDir "{app}\bin"'),
      '', SW_HIDE, ewWaitUntilTerminated, ExitCode) or (ExitCode <> 0) then begin
      Log('Falha ao remover a entrada dos shims do PATH do usuário.');
      if not UninstallSilent then
        MsgBox('Se a entrada ainda estiver no PATH do usuário, remova apenas esta pasta: ' + ExpandConstant('{app}\bin'), mbInformation, MB_OK);
    end;
  end;
  if CurUninstallStep = usPostUninstall then begin
    RegDeleteValue(HKCU, 'Software\Verman', 'PathManaged');
    RegDeleteKeyIfEmpty(HKCU, 'Software\Verman');
  end;
end;

procedure CurPageChanged(CurPageID: Integer);
begin
  if CurPageID = wpFinished then begin
    if PathConfigured then
      WizardForm.FinishedLabel.Caption := 'Verman instalado. Esta pasta foi adicionada ao PATH do usuário:' + #13#10#13#10 +
        ExpandConstant('{app}\bin') + #13#10#13#10 + 'Reabra o terminal e o editor. Se ainda não funcionar, reinicie o PC. Depois, execute: verman --version'
    else
      WizardForm.FinishedLabel.Caption := 'Verman instalado. O PATH não foi configurado nesta execução.' + #13#10#13#10 +
        'Para usar a CLI, adicione esta pasta ao PATH do usuário:' + #13#10 + ExpandConstant('{app}\bin') + #13#10#13#10 +
        'Execute novamente o instalador e marque essa opção. Depois, reabra o terminal e o editor.';
  end;
end;
