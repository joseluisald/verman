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

Antes de `dev` e `build`, o script prepara o logo e verifica o SHA-256 do instalador em `assets/installers/`. Se a versão de `src/data/product.json` ainda não estiver nessa pasta, o script procura o build local em `../dist/`. Para gerar o pacote na raiz do repositório:

```powershell
.\scripts\build-windows.ps1 -Version 0.5.2
```

Para usar outra localização, informe o caminho completo e mantenha o `.sha256` ao lado:

```powershell
$env:VERMAN_INSTALLER_PATH = 'D:\Pacotes\Verman-Setup-0.5.2-windows-amd64.exe'
```

A cada versão distribuída, atualize `src/data/product.json` e copie o novo instalador e seu `.sha256` de `../dist/` para `assets/installers/`. O build gera as cópias públicas em `public/downloads/`.

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

Marque `RAILPACK_NODE_VERSION=24` e `SITE_URL` com a URL HTTPS do domínio como disponíveis durante o build. O site é gerado com o instalador versionado e o checksum.

O pacote 0.5.4 é de revisão: assinatura, validação em Windows limpo e licença comercial do compilador Inno ainda precisam ser concluídas antes de anunciar uma distribuição comercial pronta. O download na página é identificado como prévia.

## Imagens e conteúdo

- Logo: fonte original em `../desktop/frontend/src/assets/verman-logo.png`. Para builds isolados, `assets/verman-logo.png` guarda a exportação de 256 px feita por `go run ./tools/windowsicon -out <diretório-temporário>`.
- Prints: `public/screenshots/`, capturados diretamente do EXE Wails 0.5.2 em perfil isolado. Veja `proveniencia.json`.
- Não usam a prévia de navegador nem dados desenhados para parecerem capturas.
- Exemplos de versões na documentação ilustram o formato; o catálogo do Verman é a referência de disponibilidade.
- Menus, filtros, galeria e comandos para copiar usam Alpine.js. FAQ e conteúdo dos guias também funcionam sem JavaScript.
- As licenças das fontes locais acompanham o build em `licenses/fontes.txt`.
