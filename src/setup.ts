import path from 'path';
import { setup } from 'halfred-tools';
import { userConfig } from './storage';
import { vscLogger } from './common/logger';

setup({
  logsDir: path.join(process.cwd(), './logs'),
});

vscLogger.gcStart();

userConfig.setup();
