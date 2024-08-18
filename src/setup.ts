import path from 'path';
import { setup } from 'halfred-tools';
import { records, userConfig } from './storage';
import { gitInfoCache } from './storage/gitInfo';

setup({
  logsDir: path.join(process.cwd(), './logs'),
});

userConfig.setup();
records.ensureHasCacheFile();
gitInfoCache.ensureHasCacheFile();
