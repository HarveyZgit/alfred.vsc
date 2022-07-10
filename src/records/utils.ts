import { SearchListItem } from '../common/utils';
import { records } from '../storage';

export function filterRecordBySearchKey(searchKey: string) {
  const originalRecords = records.getContent();

  return searchKey
    ? originalRecords.filter(item => new RegExp(searchKey, 'i').test(item.name))
    : originalRecords;
}

export function findRecordByPath(path: string): [SearchListItem, SearchListItem[]] {
  const allRecords = records.getContent();
  const item = allRecords.find(item => item.path === path);
  return [item, allRecords];
}
