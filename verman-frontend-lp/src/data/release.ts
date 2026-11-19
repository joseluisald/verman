import { readFile } from 'node:fs/promises';
import product from './product.json';

const release: { bytes: number | null } = JSON.parse(await readFile('public/downloads/release.json', 'utf8'));

export const installerSize = release.bytes;
export const installerPath = installerSize === null ? null : `/downloads/${product.filename}`;
