/**
 * Reference from https://github.com/mohuishou/utools/blob/master/plugins/vscode/src/files.ts
 */

import { IconElement } from 'alfy';
import { readFileSync } from 'fs';
import { isNil } from 'lodash';
import { extname } from 'path';
import initSqlJs, { Database } from 'sql.js';
import { envNames, envs, __VSC_DB_CACHE__ } from './constant';
import { store } from './store';

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

const gerProjectName = (inputPath: string) => inputPath.match(/.*\/(.*?)$/)?.[1] ?? inputPath;
const getIcon = (inputPath: string): IconElement => {
  let iconFileName = 'file';

  if (inputPath.includes('remote')) {
    iconFileName = 'remote'
  } else if (!extname(inputPath)) {
    iconFileName = 'folder'
  }

  return {
    path: `./assets/${iconFileName}.png`,
  }
}

async function createDB() {
  const sqlJS = await initSqlJs();
  const dbPath = envs.get(envNames.dbPath);
  const db = new sqlJS.Database(readFileSync(dbPath));
  store.set(__VSC_DB_CACHE__, db);

  return db;
}

async function getDB() {
  const cachedDB = store.get<Database>(__VSC_DB_CACHE__);
  if (cachedDB) return cachedDB;

  const db = await createDB();
  return db;
}

export async function getFiles(length?: number) {
  const db = await getDB();

  const sql = `select value from ItemTable where key = 'history.recentlyOpenedPathsList'`;
  const results = db.exec(sql);
  const res = results[0].values.toString();
  if (!res) {
    throw new Error(
      '数据获取失败, 注意当前仅在 vscode 1.64 版本进行过测试'
    );
  }
  const data = JSON.parse(res) as Recent;
  let { entries } = data;
  if (!isNil(length)) {
    entries = entries.slice(0, length);
  }

  return entries.map((file) => {
    if (typeof file === 'string') {
      file = { fileUri: file };
    }
    const originPath = file.fileUri || file.folderUri || file.workspace?.configPath;
    const path = decodeURIComponent(originPath ?? '')
    const name = gerProjectName(path);
    const icon = getIcon(path);

    return {
      name,
      path,
      icon,
    };
  });
}

