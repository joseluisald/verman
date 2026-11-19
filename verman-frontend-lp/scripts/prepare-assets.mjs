import { readFile, writeFile, mkdir, copyFile, stat } from "node:fs/promises";
import { createHash } from "node:crypto";
import { createReadStream } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const root = fileURLToPath(new URL("../", import.meta.url));
const repo = path.resolve(root, "..");
const product = JSON.parse(
  await readFile(path.join(root, "src/data/product.json"), "utf8"),
);
const source = process.env.VERMAN_INSTALLER_PATH
  ? path.resolve(process.env.VERMAN_INSTALLER_PATH)
  : path.join(repo, "dist", product.filename);
const downloads = path.join(root, "public/downloads");
const brand = path.join(root, "public/brand");
const licenses = path.join(root, "public/licenses");
await Promise.all([
  mkdir(downloads, { recursive: true }),
  mkdir(brand, { recursive: true }),
  mkdir(licenses, { recursive: true }),
]);
const originalLogo = path.join(
  repo,
  "desktop/frontend/src/assets/verman-logo.png",
);
const exportedLogo = path.join(
  repo,
  "dist/windows/payload/assets/verman-logo.png",
);
const [originalLogoInfo, exportedLogoInfo] = await Promise.all([
  stat(originalLogo),
  stat(exportedLogo).catch(() => null),
]);
// Reutiliza a exportação de 256 px do mesmo logo, quando não está desatualizada.
const logoSource =
  exportedLogoInfo && exportedLogoInfo.mtimeMs >= originalLogoInfo.mtimeMs
    ? exportedLogo
    : originalLogo;
await copyFile(logoSource, path.join(brand, "verman-logo.png"));
const fontLicenses = await Promise.all(
  ["manrope", "dm-sans", "jetbrains-mono"].map(async (font) => {
    const text = await readFile(
      path.join(root, "node_modules/@fontsource-variable", font, "LICENSE"),
      "utf8",
    );
    return `${font}\n${"=".repeat(font.length)}\n${text}`;
  }),
);
await writeFile(path.join(licenses, "fontes.txt"), fontLicenses.join("\n\n"));

try {
  const metadata = await stat(source);
  if (!metadata.isFile())
    throw new Error("O instalador precisa ser um arquivo regular.");
  const expected = (await readFile(`${source}.sha256`, "utf8"))
    .trim()
    .split(/\s+/)[0];
  const hash = createHash("sha256");
  for await (const chunk of createReadStream(source)) hash.update(chunk);
  const sha256 = hash.digest("hex");
  if (!/^[a-f0-9]{64}$/i.test(expected) || expected.toLowerCase() !== sha256)
    throw new Error("SHA-256 do instalador não confere.");
  await copyFile(source, path.join(downloads, product.filename));
  await writeFile(
    path.join(downloads, `${product.filename}.sha256`),
    `${sha256}  ${product.filename}\n`,
  );
  await writeFile(
    path.join(downloads, "release.json"),
    JSON.stringify({ ...product, bytes: metadata.size, sha256 }, null, 2) +
      "\n",
  );
  console.log(
    `Logo e instalador ${product.version} preparados; SHA-256 conferido.`,
  );
} catch (error) {
  throw new Error(
    `Não foi possível preparar o download: ${error.message}\nGere o pacote na raiz com scripts/build-windows.ps1 -Version ${product.version} ou informe VERMAN_INSTALLER_PATH e seu arquivo .sha256.`,
  );
}
