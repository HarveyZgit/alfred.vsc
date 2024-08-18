import path from 'path';

export const storageFilesPath = {
  user: path.join(process.cwd(), '.user.config.json'),
  records: path.join(process.cwd(), '.records.cache.json'),
  gitInfoCache: path.join(process.cwd(), '.gitInfo.cache.json'),
};
