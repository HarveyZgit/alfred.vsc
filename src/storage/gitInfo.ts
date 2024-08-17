import childProcess from 'child_process';
import { vscLogger } from '../common/logger';
import { GitInfo } from '../typings/gitInfo';

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

export function generateFolderGitInfo(path: string): null | GitInfo {
  const isGitFolder = assertGitFolder(path);

  if (!isGitFolder) {
    vscLogger.info(
      ['[getFolderGitInfo]', 'not a git folder', '-', path].join(' ')
    );
    return null;
  }

  const branch = getGitBranch(path);

  return {
    branch,
  };
}

export async function generateFolderGitInfoAsync(
  path: string
): Promise<null | GitInfo> {
  const isGitFolder = assertGitFolder(path);

  if (!isGitFolder) {
    vscLogger.info(
      ['[getFolderGitInfo]', 'not a git folder', '-', path].join(' ')
    );
    return null;
  }

  const branch = await getGitBranchAsync(path);

  return {
    branch,
  };
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
