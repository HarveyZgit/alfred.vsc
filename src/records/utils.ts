import { filterRecords } from '../filter';
import { InputInfo } from '../input';
import { RecordsStorage, records } from '../storage';
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

  // 出于性能考虑，只更新搜索结果前 10 条数据的 gitInfo
  records.brushRecordsGitInfo(list.slice(0, 10));

  return list;
}

export function findRecordByPath(path: string): [RecordItem, RecordItem[]] {
  const allRecords = records.getContent('records');
  const item = allRecords.find((item) => item.path === path);
  return [item, allRecords];
}

export function filterByIgnorePatterns(recordList: RecordItem[]): {
  records: RecordItem[];
  needDeleteRecords: RecordsStorage['trash'];
} {
  const { path: pathPatterns, type: typePatterns } = records.getContent(
    'ignorePatterns'
  ) || {
    path: [],
    type: [],
  };

  const ignoreTypeRegexps = typePatterns.map((pattern) => new RegExp(pattern, 'g'));
  const ignorePathRegexps = pathPatterns.map((pattern) => new RegExp(pattern, 'g'));

  const nextRecords: RecordItem[] = [];
  const needDeleteRecords: RecordsStorage['trash'] = {};

  recordList.forEach((record) => {
    const isIgnoredByType = ignoreTypeRegexps.some((regexp) =>
      regexp.test(record.type)
    );

    const isIgnoredByPath = ignorePathRegexps.some((regexp) =>
      regexp.test(record.pathWithoutProtocol)
    );

    if (isIgnoredByType || isIgnoredByPath) {
      needDeleteRecords[record.__vsc_id__] = record;
    } else {
      nextRecords.push(record);
    }
  });

  return {
    records: nextRecords,
    needDeleteRecords,
  };
}
