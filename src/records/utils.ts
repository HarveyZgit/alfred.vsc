import { filterRecords } from '../filter';
import { InputInfo } from '../input';
import { records } from '../storage';
import { RecordItem } from '../typings/records';
import { getRecordsFromVscodeMenu } from './getRecordsFromVscodeMenu';

export function filterRecordBySearchKey(inputInfo: InputInfo) {
  const { original: originalSearchKey } = inputInfo;
  const originalRecords = records.getContent('records');

  const recordsList = originalRecords.length
    ? originalRecords
    : getRecordsFromVscodeMenu();

  const list = originalSearchKey
    ? filterRecords(inputInfo)(recordsList)
    : recordsList;

  // 出于性能考虑，只更新搜索结果的前 10 条
  records.brushRecordsGitInfo(list.slice(0, 10));

  return list;
}

export function findRecordByPath(path: string): [RecordItem, RecordItem[]] {
  const allRecords = records.getContent('records');
  const item = allRecords.find((item) => item.path === path);
  return [item, allRecords];
}
