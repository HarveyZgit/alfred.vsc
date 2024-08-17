import { Workflow } from 'halfred-tools';
import fs from 'fs';
import { vscLogger } from './logger';
import crypto from 'crypto';
import {
  entries,
  isArray,
  isNumber,
  isObject,
  isRegExp,
  isString,
} from 'lodash';
import { RecordItem, RecordType } from '../typings/records';

export function isFolderPath(path: string, removeScheme: boolean = false) {
  if (removeScheme) {
    path = removePathScheme(path);
  }
  if (fs.existsSync(path)) {
    return fs.statSync(path).isDirectory();
  }
  return false;
}

export function isRemotePath(path: string) {
  return path.includes('remote');
}

export function isFilePath(path: string, removeScheme: boolean = false) {
  return !isFolderPath(path, removeScheme) && !isRemotePath(path);
}

export function removePathScheme(inputPath: string) {
  return inputPath.replace(/^(.*?):\/\//, '');
}

export function getRecordType(inputPath: string): RecordType {
  let recordType: RecordType = 'file';

  if (inputPath.includes('remote')) {
    recordType = 'remote';
  } else if (isFolderPath(inputPath, true)) {
    recordType = 'folder';
  }

  vscLogger.info(`${recordType} -- ${inputPath}`);

  return recordType;
}

export function getIcon(inputPath: string): Workflow.Icon {
  const iconFileName = getRecordType(inputPath);

  return {
    path: `./assets/${iconFileName}.png`,
  };
}

function generateSubtitle(item: RecordItem): string {
  const subtitleArr: string[] = [item.pathWithoutProtocol];
  const moreInfo: string[] = [item.gitInfo?.branch].filter(Boolean);

  if (moreInfo.length) {
    subtitleArr.push(moreInfo.join(', '));
  }

  return subtitleArr.join(' - ');
}

export function fmtSearchList(list: RecordItem[]) {
  return list.map<Workflow.Item>((item) => ({
    title: item.name,
    subtitle: generateSubtitle(item),
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

export function genChecker<T>(target: T) {
  return (input: any) => target === input;
}

export function anyValue2String(input: any): any {
  if (isArray(input)) {
    return input.map((item) => anyValue2String(item));
  }

  if (isRegExp(input)) {
    return input.toString();
  }

  if (isObject(input)) {
    return entries(input).reduce((prev, [k, v]) => {
      return { ...prev, [k]: anyValue2String(v) };
    }, {});
  }

  return `${input}`;
}

export const fmtInput4JsonStringify: (k: string, v: any) => any = (_, v) =>
  anyValue2String(v);
