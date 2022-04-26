/**
 * Copy from https://github.com/mohuishou/utools/blob/master/plugins/vscode/src/files.ts
 */

import { readFileSync } from 'fs';
import initSqlJs from 'sql.js';

export interface Recent {
  entries: Entry[];
}

export interface Entry {
  folderUri?: string;
  workspace?: Workspace;
  label?: string;
  remoteAuthority?: string;
  fileUri?: string;
}

export interface Workspace {
  id: string;
  configPath: string;
}

const gerProjectName = (path: string) => path.match(/.*\/(.*?)$/)?.[1] ?? path;

export async function GetFiles(path: string) {
  const db = new (await initSqlJs()).Database(readFileSync(path));
  const sql = `select value from ItemTable where key = 'history.recentlyOpenedPathsList'`;
  const results = db.exec(sql);
  const res = results[0].values.toString();
  if (!res) {
    throw new Error(
      '数据获取失败, 注意当前仅在 vscode 1.64 版本进行过测试'
    );
  }
  const data = JSON.parse(res) as Recent;

  return data.entries.map((file) => {
    if (typeof file === 'string') {
      const name = gerProjectName(file);
      return {
        name,
        path: decodeURIComponent(file),
      };
    }
    let path = file.fileUri || file.folderUri || file.workspace?.configPath;
    path = decodeURIComponent(path ?? '')
    const name = gerProjectName(path);
    return { name, path };
  });
}

