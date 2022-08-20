import { get } from 'lodash';
import { SearchListItem } from '../common/utils';
import { InputInfo } from '../input';

export namespace Condition {
  export type Checker = (Target: InputInfo) => Runner;

  export type Runner = (record: SearchListItem) => boolean;
}

export const isDefaultCondition = (..._args: any[]) => true;

export function wrapChecker(
  key: keyof InputInfo,
  func: (input: any) => boolean
) {
  return (inputInfo: InputInfo) => func(get(inputInfo, key));
}
