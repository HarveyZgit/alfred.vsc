import fs from 'fs';
import { noop } from 'lodash';
import { envNames, envs } from '../common/constant';
import { getVscodeMenuItemUriPath } from './getRecordsFromVscodeMenu';
import { getIcon, SearchListItem } from '../common/utils';
import { storageFilesPath } from '../common/paths';

function parseEnv() {
  const paths = envs.get(envNames.watchDirectories) as string;
  try {
    return paths ? paths.split(',') : [];
  } catch {
    return [];
  }
}

function updateCacheFile(content: SearchListItem[]) {
  fs.writeFile(storageFilesPath.records, JSON.stringify(content), noop);
}

function getDirectoriesByPath(dirPath: string): SearchListItem[] {
  if (!fs.statSync(dirPath).isDirectory()) return [];

  const result = fs.readdirSync(dirPath, {
    encoding: 'utf-8',
    withFileTypes: true,
  });
  return result
    .filter((item) => item.isDirectory())
    .map((item) => {
      const path = getVscodeMenuItemUriPath({
        scheme: 'file',
        path: `${dirPath}/${item.name}`,
      });
      return {
        name: item.name,
        path,
        icon: getIcon(path),
        extra: {
          from: 'getRecordsFromSpecifiedDirectory',
        },
      };
    });
}

export function getRecordsFromSpecifiedDirectory(writeFile = true) {
  const paths = parseEnv();
  if (!paths.length) return [];
  const records = paths.map(getDirectoriesByPath).flat();

  writeFile && updateCacheFile(records);
  return records;
}
