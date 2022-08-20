import { cond } from 'lodash';
import { isFilePath, isFolderPath, isRemotePath } from '../../common/utils';
import {
  InputInfo,
  isAllPathType,
  isFilePathType,
  isFolderPathType,
  isRemotePathType,
} from '../../input';
import { Condition, isDefaultCondition, wrapChecker } from '../utils';

const getPathTypeCondition: Condition.Checker = cond<
  InputInfo,
  Condition.Runner
>([
  [
    wrapChecker('pathType', isRemotePathType),
    () => (record) => isRemotePath(record.path),
  ],
  [
    wrapChecker('pathType', isFolderPathType),
    () => (record) => isFolderPath(record.path, true),
  ],
  [
    wrapChecker('pathType', isFilePathType),
    () => (record) => isFilePath(record.path, true),
  ],
  [wrapChecker('pathType', isAllPathType), () => isDefaultCondition],
]);

export default getPathTypeCondition;
