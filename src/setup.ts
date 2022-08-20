import path from 'path';
import { setup } from 'halfred-tools';
import { userConfig } from './storage';

setup({
  logsDir: path.join(process.cwd(), './logs'),
});

userConfig.setup();
