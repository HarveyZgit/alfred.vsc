import { Workflow } from 'halfred-tools';
import fs from 'fs';
import { vscLogger } from './logger';
import crypto from 'crypto';

function isDirectory(input: string) {
  if (fs.existsSync(input)) {
    return fs.statSync(input).isDirectory();
  }
  return false;
}

export function getIcon(inputPath: string): Workflow.Icon {
  const pathWithoutScheme = inputPath.replace(/^(.*?):\/\//, '');
  let iconFileName = 'file';

  if (inputPath.includes('remote')) {
    iconFileName = 'remote';
  } else if (isDirectory(pathWithoutScheme)) {
    iconFileName = 'folder';
  }

  vscLogger.info(`${iconFileName} -- ${inputPath}`);

  return {
    path: `./assets/${iconFileName}.png`,
  };
}

export interface SearchListItem {
  __vsc_id__: string;
  name: string;
  path: string;
  icon: Workflow.Icon;
  extra?: {
    from: string;
    [x: string]: any;
  };
}

export function fmtSearchList(list: SearchListItem[]) {
  return list.map<Workflow.Item>((item) => ({
    title: item.name,
    subtitle: item.path,
    arg: item.path,
    icon: item.icon,
  }));
}

export function genRecordId(str: string) {
  if (isRecordId(str)) return str;
  return crypto.createHash('md5').update(str).digest('hex');
}

export function isRecordId(str: string) {
  // 用是否存在 path 标识来判断是否为 record id
  return str.indexOf('://') < 0;
}
