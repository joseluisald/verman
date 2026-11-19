# Verman — landing page e documentação

Astro + Tailwind CSS + Alpine.js (build CSP). Site estático, responsivo e em português, com fontes locais e capturas reais do aplicativo Windows. Os guias ficam em `src/data/docs.ts`.

## Rodar

Use Node 24 LTS e npm. Dentro desta pasta:

```powershell
npm ci --ignore-scripts
npm run dev
```

Abra o endereço local exibido no terminal, normalmente `http://127.0.0.1:4321`.

```powershell
npm run build
npm run preview
```

O site pronto fica em `dist/` desta pasta. O build verifica os tipos antes de gerar as páginas.

## Download do instalador

Antes de `dev` e `build`, o script prepara o logo e o instalador, conferindo o arquivo `.sha256`. Por padrão usa `../dist/Verman-Setup-0.5.2-windows-amd64.exe`. Gere esse pacote na raiz do repositório, se ainda não existir:

```powershell
.\scripts\build-windows.ps1 -Version 0.5.2
```

Para usar outra localização, informe o caminho completo e mantenha o `.sha256` ao lado:

```powershell
$env:VERMAN_INSTALLER_PATH = 'D:\Pacotes\Verman-Setup-0.5.2-windows-amd64.exe'
```

Os downloads e a cópia do logo em `public/` são gerados e ignorados pelo Git; não são placeholders. O build os inclui no site, de modo que os botões de download funcionam localmente. Atualize `src/data/product.json` ao mudar a versão distribuída.

## Deploy no Coolify com Railpack

Selecione o repositório e a branch que contém a LP. O Railpack pode usar só esta pasta como contexto de build.

| Campo | Valor |
| --- | --- |
| Build strategy | Railpack |
| Site type | Static |
| Base directory | `/verman-frontend-lp` |
| Publish directory | `/dist` |
| Web server e porta | `nginx:alpine`, porta `80` |
| Install command | `npm ci --ignore-scripts` |
| Build command | `npm run build` |

Marque `RAILPACK_NODE_VERSION=24`, `SITE_URL` com a URL HTTPS do domínio, e `VERMAN_SITE_ONLY=1` como disponíveis durante o build. Em um clone do Git, o pacote Windows local está ausente. Nesse modo, o site é gerado com os guias, as capturas e o logo, e sem publicar links de download ou checksum.

Para publicar também o download, disponibilize o instalador e o respectivo `.sha256` em um diretório acessível durante o build, defina `VERMAN_INSTALLER_PATH` com o caminho do instalador nesse ambiente e remova `VERMAN_SITE_ONLY`. O build verifica o checksum antes de copiar o executável para a saída estática.

O pacote 0.5.2 é de revisão: assinatura, validação em Windows limpo e licença comercial do compilador Inno ainda precisam ser concluídas antes de anunciar uma distribuição comercial pronta. O download na página é identificado como prévia.

## Imagens e conteúdo

- Logo: fonte original em `../desktop/frontend/src/assets/verman-logo.png`. Para builds isolados, `assets/verman-logo.png` guarda a exportação de 256 px feita por `go run ./tools/windowsicon -out <diretório-temporário>`.
- Prints: `public/screenshots/`, capturados diretamente do EXE Wails 0.5.2 em perfil isolado. Veja `proveniencia.json`.
- Não usam a prévia de navegador nem dados desenhados para parecerem capturas.
- Exemplos de versões na documentação ilustram o formato; o catálogo do Verman é a referência de disponibilidade.
- Menus, filtros, galeria e comandos para copiar usam Alpine.js. FAQ e conteúdo dos guias também funcionam sem JavaScript.
- As licenças das fontes locais acompanham o build em `licenses/fontes.txt`.
