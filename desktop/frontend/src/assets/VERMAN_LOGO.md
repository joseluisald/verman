# Logo Verman

Arquivo: `verman-logo.png`, PNG de 1254 × 1254 pixels, fundo verde escuro opaco.

O V representa Verman; a faixa paralela remete às versões e o traço inferior ao cursor de terminal. Aplicado à marca da interface e ao favicon. O arquivo é raster, não um SVG vetorizado. Cores de referência: verde escuro `#173B32` e verde menta `#BCE5CE`.

O build Windows usa `tools/windowsicon` para gerar `dist/windows/payload/assets/verman.ico` (16, 24, 32, 48, 64, 128 e 256 px), a imagem do cabeçalho e o painel do instalador. Incorpora o ícone ao EXE do desktop e instala o ICO usado pelos atalhos do menu Iniciar e da área de trabalho. Para substituir a identidade, atualize o PNG original e execute novamente o build completo.

Gerado com a ferramenta integrada `image_gen` (skill imagegen). Prompt final:

```text
Final production application logo for Verman. Deliver a single perfectly square opaque app icon image. The entire canvas is a solid uniform flat dark forest green #173B32 background, reaching all four edges. NO transparency anywhere, no black patches, no gradients, no glows, no shadows, no textures, no lighting. Center a large crisp mint green #BCE5CE geometric V symbol with generous 15 percent padding. The V has broad angular diagonal arms, a wide flat truncated bottom, and a visually distinctive thin parallel mint ribbon ABOVE AND ALONG THE LEFT ARM, separated from the left arm by a narrow uniform dark green gap. This second inset diagonal left ribbon starts with a short horizontal cap, suggesting a stack of software versions. Add one small mint horizontal terminal cursor to the lower right of the V, clearly separated, aligned to the bottom. Flat two-color vector-like visual, sophisticated minimal developer tooling brand, optically balanced, compact strong silhouette recognizable at 24px. All mint regions same flat solid color. No rounded-square tile borders, no extra framing, no letters or words or typography except this abstract V monogram, no mockup, no contact sheet. Only one final logo asset. Fully opaque dark green background.
```
