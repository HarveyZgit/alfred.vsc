import * as path from 'path';
import json from '@rollup/plugin-json';
import copy from 'rollup-plugin-copy';
import rollupTypescript from 'rollup-plugin-typescript2';
import replacement from 'rollup-plugin-module-replacement';

const root = path.join(__dirname, '../');
const buildTsConfigPath = path.join(root, 'tsconfig.build.json');

export const plugins = [
  json(),
  rollupTypescript({
    tsconfig: buildTsConfigPath,
  }),
  replacement({
    entries: [
      {
        find: 'sqljs',
        replacement: path.join(root, 'third-party/sqljs/sql-wasm.js'),
      },
    ],
  }),
  copy({
    targets: [
      { src: 'public/*', dest: 'build' },
      { src: 'README.md', dest: 'build' },
      { src: 'assets', dest: 'build' },
      { src: ['package.json', 'pnpm-lock.yaml'], dest: 'build' },
      { src: 'third-party/sqljs/sql-wasm.wasm', dest: 'build/lib' },
    ],
  }),
];

export const external = [
  'halfred-tools',
  'fs',
  'lodash',
  'os',
  'path',
  'commander',
  'crypto',
];

export function input(entry) {
  return path.join(__dirname, '../', entry);
}
