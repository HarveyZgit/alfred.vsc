import childProcess from 'child_process';
import { vscLogger } from '../common/logger';
import { GitInfo } from '../typings/gitInfo';
import { readFile, writeFile } from './utils';
import { storageFilesPath } from '../common/paths';
import { get, isFunction, set } from 'lodash';
import { RecordItem } from '../typings/records';

function assertSpawnSyncError(
  result: ReturnType<typeof childProcess.spawnSync>
): boolean {
  const { stderr, error } = result;
  return Boolean(error || stderr.toString('utf8'));
}

function assertGitFolder(path: string): boolean {
  const result = childProcess.spawnSync('git', ['status'], {
    cwd: path,
  });

  const hasError = assertSpawnSyncError(result);

  return !hasError;
}

export function generateFolderGitInfo(record: RecordItem): null | GitInfo {
  const path = record.pathWithoutProtocol;
  const isGitFolder = assertGitFolder(path);

  if (!isGitFolder) {
    vscLogger.info(
      ['[getFolderGitInfo]', 'not a git folder', '-', path].join(' ')
    );
    return null;
  }

  const cache = gitInfoCache.getCachedGitInfo(record.__vsc_id__);
  if (cache) {
    vscLogger.info(
      [
        '[generateFolderGitInfoAsync]',
        'using gitInfo cache',
        '-',
        path,
        '-',
        record.__vsc_id__,
      ].join(' ')
    );
    return cache;
  }

  const branch = getGitBranch(path);

  const data = {
    branch,
  };

  gitInfoCache.update(record.__vsc_id__, data, true);

  return data;
}

/** @deprecated */
export async function generateFolderGitInfoAsync(
  record: RecordItem
): Promise<null | GitInfo> {
  const path = record.pathWithoutProtocol;
  const isGitFolder = assertGitFolder(path);

  if (!isGitFolder) {
    vscLogger.info(
      ['[getFolderGitInfo]', 'not a git folder', '-', path].join(' ')
    );
    return null;
  }

  const cache = gitInfoCache.getCachedGitInfo(record.__vsc_id__);
  if (cache) {
    vscLogger.info(
      [
        '[generateFolderGitInfoAsync]',
        'using gitInfo cache',
        '-',
        path,
        '-',
        record.__vsc_id__,
      ].join(' ')
    );
    return cache;
  }

  const branch = await getGitBranchAsync(path);

  const data = {
    branch,
  };

  gitInfoCache.update(record.__vsc_id__, data, true);

  return data;
}

function getGitBranch(path: string): string {
  const result = childProcess.spawnSync(
    'git',
    ['rev-parse', '--abbrev-ref', 'HEAD'],
    {
      cwd: path,
    }
  );

  if (assertSpawnSyncError(result)) {
    return '';
  }

  return result.stdout.toString('utf8').replace(/\n/g, '');
}

/** @deprecated */
function getGitBranchAsync(path: string): Promise<string> {
  return new Promise((resolve) => {
    const terminal = childProcess.spawn(
      'git',
      ['rev-parse', '--abbrev-ref', 'HEAD'],
      {
        cwd: path,
      }
    );

    terminal.stdout.on('data', (data) => {
      const branch = data.toString('utf8').replace(/\n/g, '');
      resolve(branch);
    });

    terminal.stderr.on('data', () => {
      resolve('');
    });
  });
}

type GitInfoCacheContent = { ts: number; gitInfo: GitInfo | null };
type GitInfoCacheObject = Record<string, GitInfoCacheContent>;

export const gitInfoCache = {
  filePath: storageFilesPath.gitInfoCache,

  get fillContent(): GitInfoCacheObject {
    return readFile(
      gitInfoCache.filePath,
      true,
      gitInfoCache.getDefaultContent()
    );
  },

  getDefaultContent: (): GitInfoCacheObject => ({}),

  getDefaultItemContent: (): GitInfoCacheContent => ({
    ts: Date.now(),
    gitInfo: null,
  }),

  ensureHasCacheFile: () => {
    try {
      readFile(gitInfoCache.filePath, true);
    } catch {
      // 同步写一个文件，防止初始化时其他搜索操作报错
      writeFile(gitInfoCache.filePath, gitInfoCache.getDefaultContent(), true);
    }
  },

  update: (
    recordId: string,
    content: GitInfo | ((prevContent: GitInfo) => GitInfo),
    sync = false
  ) => {
    const prevContent = readFile<GitInfoCacheObject>(
      gitInfoCache.filePath,
      true,
      gitInfoCache.getDefaultContent()
    );

    const inputContent = isFunction(content)
      ? content(get(prevContent, recordId).gitInfo)
      : content;
    const nextContent = set(prevContent, recordId, {
      ts: Date.now(),
      gitInfo: inputContent,
    });
    writeFile(gitInfoCache.filePath, nextContent, sync);
  },

  getItem: (recordId: string): GitInfoCacheContent => {
    return get(
      gitInfoCache.fillContent,
      recordId,
      gitInfoCache.getDefaultItemContent()
    );
  },

  getCachedGitInfo: (recordId: string): GitInfo => {
    const cache = gitInfoCache.getItem(recordId);
    if (gitInfoCache.isExpired(cache.ts)) {
      return null;
    }
    return cache.gitInfo;
  },

  isExpired: (ts: number) => {
    return Date.now() - ts > 30 * 1000;
  },
};
