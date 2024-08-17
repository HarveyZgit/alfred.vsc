import fs from 'fs';
import { get, has, isFunction, keys, merge, noop, set, uniqBy } from 'lodash';
import { storageFilesPath } from '../common/paths';
import { getRecordsFromSpecifiedDirectory } from '../records/getRecordsFromSpecifiedDirectory';
import { getRecordsFromVscodeDB } from '../records/getRecordsFromVscodeDB';
import { getRecordsFromVscodeMenu } from '../records/getRecordsFromVscodeMenu';
import { RecordItem } from '../typings/records';
import { getRecordType, removePathScheme } from '../common/utils';
import { generateFolderGitInfo } from './gitInfo';
import { vscLogger } from '../common/logger';

export interface RecordsStorage {
  records: RecordItem[];
  trash: Record<string, RecordItem>;
  ignorePatterns: {
    path: string[];
    type: string[];
  };
}

const getRecordsStorageDefaultValue: () => RecordsStorage = () => ({
  records: [],
  trash: {},
  ignorePatterns: {
    path: [],
    type: [],
  },
});

function readFile<T>(path: string, parse: true, defaultValue?: T): T;
function readFile<T>(path: string, parse: false, defaultValue?: T): string;
function readFile<T>(path: string, parse: boolean, defaultValue?: T) {
  const content = fs.readFileSync(path, { encoding: 'utf-8' });
  if (parse) {
    try {
      return JSON.parse(content) as T;
    } catch (error) {
      return defaultValue;
    }
  }

  return content;
}

function writeFile<T>(path: string, content: T, sync: boolean) {
  const write = sync ? fs.writeFileSync : fs.writeFile;
  write.call(
    fs,
    path,
    JSON.stringify(content, null, 2),
    sync ? undefined : noop
  );
}

export const records = {
  filePath: storageFilesPath.records,

  get fillContent(): RecordsStorage {
    return readFile(records.filePath, true, records.getDefaultContent());
  },

  getDefaultContent(custom?: RecordsStorage): RecordsStorage {
    return merge(getRecordsStorageDefaultValue(), custom);
  },

  update<T extends keyof RecordsStorage>(
    key: T,
    content:
      | RecordsStorage[T]
      | ((prevContent: RecordsStorage[T]) => RecordsStorage[T]),
    sync = false
  ) {
    const prevContent = readFile<RecordsStorage>(
      records.filePath,
      true,
      records.getDefaultContent()
    );

    const inputContent = isFunction(content)
      ? content(
          get(prevContent, key, get(getRecordsStorageDefaultValue(), key))
        )
      : content;
    const nextContent = set(prevContent, key, inputContent);
    writeFile(records.filePath, nextContent, sync);
  },

  replaceAll: (content: RecordsStorage, sync = false) => {
    writeFile(records.filePath, content, sync);
  },

  getContent<T extends keyof RecordsStorage>(key: T): RecordsStorage[T] {
    return get(records.fillContent, key, get(records.getDefaultContent(), key));
  },

  getAllContent: (): RecordsStorage => records.fillContent,

  /** 确保有一个缓存文件 */
  ensureHasCacheFile: () => {
    try {
      readFile(records.filePath, true);
    } catch {
      // 同步写一个文件，防止初始化时其他搜索操作报错
      records.replaceAll(
        records.getDefaultContent({
          records: getRecordsFromVscodeMenu(),
          trash: {},
          ignorePatterns: {
            path: [],
            type: [],
          },
        }),
        true
      );
    }
  },

  brushRecords: (recordList: RecordItem[]) => {
    recordList.forEach((record) => {
      /** 刷数：添加 type */
      if (!has(record, 'type')) {
        set(record, 'type', getRecordType(record.path));
      }

      /** 刷数：添加 pathWithoutProtocol */
      if (!has(record, 'pathWithoutProtocol')) {
        set(record, 'pathWithoutProtocol', removePathScheme(record.path));
      }
    });
  },

  brushRecordsGitInfo: (recordList: RecordItem[]) => {
    recordList.forEach((record) => {
      /** 刷数：添加 gitInfo */
      if (record.type === 'folder') {
        const gitInfo = generateFolderGitInfo(record.pathWithoutProtocol);

        vscLogger.info(
          [
            '[brushRecords]',
            'gitInfo',
            '-',
            record.pathWithoutProtocol,
            gitInfo ? JSON.stringify(gitInfo) : null,
          ].join(' ')
        );

        set(record, 'gitInfo', gitInfo);
      }
    });
  },

  setup: async (dropAll?: boolean) => {
    // 初始化 .records.cache.json
    // 1-1. 从 vscode db 中获取记录
    // 1-2. 从用户指定目录获取记录
    // 2. 合并二者的目录，去重
    // 3. 写入 .records.cache.json

    records.ensureHasCacheFile();

    const initialRecords = await Promise.all([
      getRecordsFromSpecifiedDirectory(false),
      getRecordsFromVscodeDB(100), // 只取最近用的 100 条应该就够了
    ]);

    let uniqRecords = uniqBy(initialRecords.flat(), (record) => record.path);

    records.brushRecords(uniqRecords);

    if (dropAll) {
      records.update('trash', {}, true);
    } else {
      const prevTrash = records.getContent('trash') ?? {};
      const prevTrashIds = keys(prevTrash);

      uniqRecords = uniqRecords.filter(
        (item) => !prevTrashIds.includes(item.__vsc_id__)
      );
    }

    records.update('records', uniqRecords, true);
  },
};

interface UserConfig {
  hasInitial: boolean;
}

export const userConfig = {
  filePath: storageFilesPath.user,

  update: (content: Partial<UserConfig>) => {
    const prevConfig = userConfig.getContent();
    const nextConfig = merge(prevConfig ?? {}, content);

    try {
      writeFile(userConfig.filePath, nextConfig, true);

      return { success: true, config: nextConfig };
    } catch {
      return { success: false, config: prevConfig };
    }
  },

  getContent: (): UserConfig => {
    try {
      const content = fs.readFileSync(userConfig.filePath, {
        encoding: 'utf-8',
      });
      return JSON.parse(content);
    } catch {
      return null;
    }
  },

  setup: () => {
    const userConf = userConfig.getContent();
    if (userConf && userConf.hasInitial) {
      return;
    }
    records.setup();
    userConfig.update({
      hasInitial: true,
    });
  },
};
