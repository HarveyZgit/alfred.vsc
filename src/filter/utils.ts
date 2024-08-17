import { get } from 'lodash';
import { InputInfo } from '../input';
import { RecordItem } from '../typings/records';

export namespace Condition {
  export type Checker = (Target: InputInfo) => Runner;

  export type Runner = (record: RecordItem) => boolean;
}

export const isDefaultCondition = (..._args: any[]) => true;

export function wrapChecker(
  key: keyof InputInfo,
  func: (input: any) => boolean
) {
  return (inputInfo: InputInfo) => func(get(inputInfo, key));
}
