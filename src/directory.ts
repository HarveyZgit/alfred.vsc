import fs from 'fs';
import { noop } from 'lodash';
import path from 'path';
import { envNames, envs } from './constant';
import { getIcon, SearchListItem } from './utils';

const RECORD_CACHE_FILE = path.join(__dirname, '.records.cache.json');

function parseEnv() {
  const paths = envs.get(envNames.watchDirectories) as string;
  try {
    return paths.split(',');
  } catch {
    return [];
  }
}

function updateCacheFile(content: SearchListItem[]) {
  fs.writeFile(
    RECORD_CACHE_FILE,
    JSON.stringify(content),
    noop,
  );
}

function getDirectoriesByPath(dirPath: string): SearchListItem[] {
  if (!fs.statSync(dirPath).isDirectory()) return [];

  const result = fs.readdirSync(dirPath, { encoding: 'utf-8', withFileTypes: true });
  return result
    .filter(item => item.isDirectory())
    .map(item => {
      const path = `file://${dirPath}/${item.name}`;
      return {
        name: item.name,
        path,
        icon: getIcon(path),
      }
    });
}

export function updateDirectoryRecords() {
  const paths = parseEnv();
  const records = paths.map(getDirectoriesByPath).flat();

  updateCacheFile(records);
  return records;
}

function main(): SearchListItem[] {
  try {
    const content = fs.readFileSync(RECORD_CACHE_FILE, { encoding: 'utf-8' });
    return JSON.parse(content);
  } catch {
    return updateDirectoryRecords();
  }
}

export default main;
main();
