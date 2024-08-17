import fs from 'fs';
import os from 'os';
import { noop } from 'lodash';
import { envNames, envs } from '../common/constant';
import { getVscodeMenuItemUriPath } from './getRecordsFromVscodeMenu';
import { genRecordId, getIcon, getRecordType } from '../common/utils';
import { storageFilesPath } from '../common/paths';
import { RecordItem } from '../typings/records';
import { generateFolderGitInfo } from '../storage/gitInfo';
import { filterByIgnorePatterns } from './utils';

function parseEnv() {
  const paths = envs.get(envNames.watchDirectories) as string;
  try {
    return paths ? paths.split(',') : [];
  } catch {
    return [];
  }
}

function updateCacheFile(content: RecordItem[]) {
  fs.writeFile(storageFilesPath.records, JSON.stringify(content), noop);
}

function getDirectoriesByPath(dirPath: string): RecordItem[] {
  if (dirPath.startsWith('~')) {
    dirPath = dirPath.replace('~', os.homedir());
  }
  if (!fs.statSync(dirPath).isDirectory()) return [];

  const result = fs.readdirSync(dirPath, {
    encoding: 'utf-8',
    withFileTypes: true,
  });
  const list = result
    .filter((item) => item.isDirectory())
    .map((item) => {
      const pathWithoutProtocol = `${dirPath}/${item.name}`;
      const path = getVscodeMenuItemUriPath({
        scheme: 'file',
        path: pathWithoutProtocol,
      });
      const gitInfo = generateFolderGitInfo(pathWithoutProtocol);

      return {
        __vsc_id__: genRecordId(path),
        name: item.name,
        path,
        pathWithoutProtocol,
        type: getRecordType(path),
        icon: getIcon(path),
        gitInfo,
        extra: {
          from: 'getRecordsFromSpecifiedDirectory',
        },
      };
    });

  return filterByIgnorePatterns(list).records;
}

export function getRecordsFromSpecifiedDirectory(writeFile = true) {
  const paths = parseEnv();
  if (!paths.length) return [];
  const records = paths.map(getDirectoriesByPath).flat();

  writeFile && updateCacheFile(records);
  return records;
}
