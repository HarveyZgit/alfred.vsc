import { SearchListItem } from '../common/utils';
import { records } from '../storage';
import { getRecordsFromVscodeMenu } from './getRecordsFromVscodeMenu';

export function filterRecordBySearchKey(searchKey: string) {
  const originalRecords = records.getContent('records');

  const recordsList = originalRecords.length
    ? originalRecords
    : getRecordsFromVscodeMenu();

  return searchKey
    ? recordsList.filter((item) => new RegExp(searchKey, 'i').test(item.name))
    : recordsList;
}

export function findRecordByPath(
  path: string
): [SearchListItem, SearchListItem[]] {
  const allRecords = records.getContent('records');
  const item = allRecords.find((item) => item.path === path);
  return [item, allRecords];
}
