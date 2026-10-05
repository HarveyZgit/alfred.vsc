import { api, token } from './api.js';
import { $, notice, action } from './dom.js';
import { records, bindRecords } from './records.js';
import { settings, bindSettings } from './settings.js';
import { migrations, bindMigration } from './migration.js';

for (const tab of document.querySelectorAll('[data-tab]')) {
  tab.addEventListener(
    'click',
    action(async () => {
      document
        .querySelectorAll('[data-tab]')
        .forEach((t) => t.classList.toggle('active', t === tab));
      for (const name of ['records', 'settings', 'migration']) {
        $(name).hidden = name !== tab.dataset.tab;
      }
      if (tab.dataset.tab === 'records') {
        await records();
      }
      if (tab.dataset.tab === 'settings') {
        await settings();
      }
      if (tab.dataset.tab === 'migration') {
        await migrations();
      }
    }),
  );
}
bindRecords();
bindSettings();
bindMigration();

$('close').onclick = action(async () => {
  await api('close', {});
  document.querySelectorAll('button,input,textarea,select').forEach((e) => (e.disabled = true));
  notice('面板服务已关闭，可以关闭此标签页。');
});

$('version').textContent = 'VSC 3';
if (!token) {
  notice('请从 Alfred 的管理入口重新打开此面板。', true);
} else {
  records().catch((e) => notice(e.message, true));
}
