import { every, filter } from 'lodash';
import { InputInfo } from '../input';
import { getPathTypeCondition, getSearchKeyCondition } from './conditions';
import { Condition } from './utils';
import { RecordItem } from '../typings/records';

/**
 * 在这里添加需要用到的 condition checkers
 */
const checkers: Condition.Checker[] = [
  getPathTypeCondition,
  getSearchKeyCondition,
];

const genCheckers = (inputInfo: InputInfo) =>
  checkers.map((item) => item(inputInfo));

export function filterRecords(inputInfo: InputInfo) {
  const conditionCheckers: Condition.Runner[] = genCheckers(inputInfo);

  return (records: RecordItem[]) => {
    return filter(records, (record) => {
      return every(conditionCheckers, (checker) => checker(record));
    });
  };
}
