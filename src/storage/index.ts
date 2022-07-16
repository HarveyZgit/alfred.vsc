import fs from 'fs';
import { merge, noop, uniqBy } from 'lodash';
import { storageFilesPath } from '../common/paths';
import { SearchListItem } from '../common/utils';
import { getRecordsFromSpecifiedDirectory } from '../records/getRecordsFromSpecifiedDirectory';
import { getRecordsFromVscodeDB } from '../records/getRecordsFromVscodeDB';
import { getRecordsFromVscodeMenu } from '../records/getRecordsFromVscodeMenu';

function writeFile<T>(path: string, content: T, sync: boolean) {
  const write = sync ? fs.writeFileSync : fs.writeFile;
  write.call(
    fs,
    path,
    JSON.stringify(content, null, 2),
    sync ? undefined : noop,
  );
}

export const records = {
  filePath: storageFilesPath.records,

  update: (content: SearchListItem[], sync = false) => {
    writeFile(records.filePath, content, sync);
  },

  getContent: (): SearchListItem[] => {
    const userConf = userConfig.getContent();

    // 第一次获取内容的时候，自动初始化
    if (!userConf?.hasInitial) {
      userConfig.setup();
      return getRecordsFromVscodeMenu();
    }

    try {
      const content = fs.readFileSync(records.filePath, { encoding: 'utf-8' });
      return JSON.parse(content);
    } catch {
      const fallbackRecords = getRecordsFromVscodeMenu();
      // TODO 执行更新规则
      records.update(fallbackRecords);
      return fallbackRecords;
    }
  },

  setup: async () => {
    // 初始化 .records.cache.json
    // 1-1. 从 vscode db 中获取记录
    // 1-2. 从用户指定目录获取记录
    // 2. 合并二者的目录，去重
    // 3. 写入 .records.cache.json

    // 先同步写一个文件，防止初始化时其他搜索操作报错
    records.update([], true);

    const initialRecords = await Promise.all([
      getRecordsFromSpecifiedDirectory(false),
      getRecordsFromVscodeDB(100), // 只取最近用的 100 条应该就够了
    ]);

    const uniqRecords = uniqBy(initialRecords.flat(), record => record.path);

    records.update(uniqRecords, true);
  },
}

interface UserConfig {
  hasInitial: boolean;
}

export const userConfig = {
  filePath: storageFilesPath.user,

  update: (content: Partial<UserConfig>) => {
    const prevConfig = userConfig.getContent();
    const nextConfig = merge(prevConfig ?? {}, content);

    try {
      writeFile(
        userConfig.filePath,
        nextConfig,
        true,
      )

      return { success: true, config: nextConfig };
    } catch {
      return { success: false, config: prevConfig };
    }
  },

  getContent: (): UserConfig => {
    try {
      const content = fs.readFileSync(userConfig.filePath, { encoding: 'utf-8' });
      return JSON.parse(content);
    } catch {
      return null;
    }
  },

  setup: () => {
    records.setup();
    userConfig.update({
      hasInitial: true,
    });
  },
}
