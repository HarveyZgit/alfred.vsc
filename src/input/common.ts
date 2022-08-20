import { genChecker } from '../common/utils';

export interface InputInfo {
  original: string;
  pathType: PathType;
  searchKey: string;
  searchKeyReg: RegExp;
}

export enum PathType {
  file = 'file',
  folder = 'folder',
  remote = 'remote',
  all = 'all',
}

export const pathTypeAliases = {
  [PathType.file]: ['file', 'fi', 'f'],
  [PathType.folder]: ['folder', 'fo', 'ff', 'dir', 'd'],
  [PathType.remote]: ['remote', 're', 'r'],
};

export const isFilePathType = genChecker(PathType.file);
export const isFolderPathType = genChecker(PathType.folder);
export const isRemotePathType = genChecker(PathType.remote);
export const isAllPathType = genChecker(PathType.all);
