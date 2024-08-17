import { entries } from 'lodash';
import { InputInfo, PathType, pathTypeAliases } from './common';

/**
 * 对 input 的解析原则
 *
 * 1. 只对 input 中传入的第一个参数进行解析
 * 2. 这个参数可能是:
 *   - pathType: path 的类型
 *   - command: 执行某个命令（比如 rm - 删除）
 *   - ...
 */
export const parser = (original: string = ''): InputInfo => {
  const params = original.split(' ');
  const searchKey = getSearchKey(params).join(' ');
  const pathType = getSpecifyPathType(params);

  return {
    original,
    pathType,
    searchKey,
    searchKeyReg: new RegExp(searchKey, 'i'),
  };
};

function getSearchKey(params: string[]) {
  if (params.length === 1) return params;
  const [, ...rest] = params;
  return rest;
}

function getSpecifyPathType(params: string[]): PathType {
  if (params.length === 1) return PathType.all;

  const [maybePathType] = params;

  const aliasEntries = entries(pathTypeAliases);

  for (let i = 0; i < aliasEntries.length; i++) {
    const [pathType, aliases] = aliasEntries[i];
    if (aliases.includes(maybePathType)) return pathType as PathType;
  }

  return PathType.all;
}
