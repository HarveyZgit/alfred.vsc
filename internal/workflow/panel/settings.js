import { api } from './api.js';
import { $, editors, element, notice, action } from './dom.js';
import { invalidatePreview } from './migration.js';

async function settings() {
  const data = await api('settings');
  $('roots').value = (data.roots || []).join('\n');
  $('default-editor').value = data.editor;
  $('editor-paths').replaceChildren();
  for (const [key, label] of Object.entries(editors)) {
    const wrapper = element('div');
    const l = element('label', label);
    l.htmlFor = 'editor-' + key;
    const input = element('input');
    input.id = 'editor-' + key;
    input.value = data.editors[key] || '';
    wrapper.append(l, input);
    $('editor-paths').append(wrapper);
  }
  $('settings-mode').textContent = data.managed
    ? '当前使用面板设置，优先于 Alfred 变量和 config.json 顶层同名项。'
    : '当前使用 Alfred 变量 / config.json；保存后，这里的根目录与 IDE 设置优先生效。';
}

function bindSettings() {
  $('settings-form').onsubmit = action(async (event) => {
    event.preventDefault();
    const paths = {};
    for (const key of Object.keys(editors)) {
      paths[key] = $('editor-' + key).value.trim();
    }
    await api('settings', {
      roots: $('roots')
        .value.split('\n')
        .map((s) => s.trim())
        .filter(Boolean),
      editor: $('default-editor').value,
      editors: paths,
    });
    invalidatePreview();
    await settings();
    notice('配置已保存，下次 Alfred 查询即生效。修改了 Git 根目录时，请到项目记录刷新索引。');
  });
  $('reset-settings').onclick = action(async () => {
    await api('settings/reset', {});
    invalidatePreview();
    await settings();
    notice('已恢复使用 Alfred 变量和高级配置。');
  });
}

export { settings, bindSettings };
