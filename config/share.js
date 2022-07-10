import * as path from 'path';
import json from '@rollup/plugin-json';
import rollupTypescript from 'rollup-plugin-typescript2';


const buildTsConfigPath = path.join(__dirname, '../tsconfig.build.json');

export const plugins = [
  json(),
  rollupTypescript({
    tsconfig: buildTsConfigPath,
  }),
];

export const external = ['alfy', 'fs', 'sql.js', 'lodash', 'os', 'path', 'commander'];

export function input(entry) {
  return path.join(__dirname, '../', entry);
}
