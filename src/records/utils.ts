import { SearchListItem } from '../common/utils';
import { filterRecords } from '../filter';
import { InputInfo } from '../input';
import { records } from '../storage';
import { getRecordsFromVscodeMenu } from './getRecordsFromVscodeMenu';

export function filterRecordBySearchKey(inputInfo: InputInfo) {
  const { original: originalSearchKey } = inputInfo;
  const originalRecords = records.getContent('records');

  const recordsList = originalRecords.length
    ? originalRecords
    : getRecordsFromVscodeMenu();

  return originalSearchKey
    ? filterRecords(inputInfo)(recordsList)
    : recordsList;
}

export function findRecordByPath(
  path: string
): [SearchListItem, SearchListItem[]] {
  const allRecords = records.getContent('records');
  const item = allRecords.find((item) => item.path === path);
  return [item, allRecords];
}
