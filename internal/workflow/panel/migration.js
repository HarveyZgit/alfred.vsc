import { api } from './api.js';
import { $, element, notice, action, button } from './dom.js';

let previewSource = '';

function renderReport(report, historical = false) {
  const box = element('div', undefined, 'card');
  box.append(
    element(
      'p',
      historical
        ? '首次迁移时的快照（可能过期，请重新预览）'
        : report.applied
          ? '最近一次执行结果（当前范围请重新预览）'
          : '当前迁移预览',
      'hint',
    ),
  );
  box.append(
    element(
      'p',
      `范围内目录 ${report.directory_order_imported} / ${report.records} · 范围内隐藏项 ${report.hidden_imported} / ${report.hidden_records} · 未迁移 ${(report.unmigrated || []).length}`,
      'report-summary',
    ),
  );
  box.append(element('p', '来源：' + report.source, 'path'));
  if (report.applied) {
    box.append(
      element(
        'p',
        report.already_applied ? '这些记录已处理，无需重复写入。' : '迁移已完成。',
        'hint',
      ),
    );
  }
  if (report.notes?.length) {
    box.append(element('p', report.notes.join('\n'), 'hint'));
  }
  for (const item of report.unmigrated || []) {
    const row = element('div', undefined, 'migration-item');
    row.append(element('p', item.original), element('p', item.how_to_migrate, 'hint'));
    box.append(row);
  }
  return box;
}

async function migrations() {
  const data = await api('migrations');
  $('migration-history').replaceChildren();
  for (const item of data) {
    const card = renderReport(item.report, item.historical);
    card.append(
      button('使用此来源重新预览', async () => {
        $('source').value = item.report.source;
        await preview();
      }),
    );
    $('migration-history').append(card);
  }
  if (!data.length) {
    $('migration-history').append(element('p', '还没有迁移记录。', 'hint'));
  }
}

async function preview() {
  previewSource = '';
  $('apply-migration').disabled = true;
  notice('正在按当前来源检查迁移范围…');
  const source = $('source').value.trim();
  const report = await api('migrate', {
    source,
    apply: false,
  });
  $('migration-result').replaceChildren(renderReport(report));
  previewSource = source;
  $('apply-migration').disabled = report.already_applied;
  notice(
    report.already_applied
      ? '当前范围已迁移，无需重复执行。'
      : '预览完成。确认列表后可执行补迁移。',
  );
}

function invalidatePreview() {
  previewSource = '';
  $('apply-migration').disabled = true;
}

function bindMigration() {
  $('source').oninput = () => {
    previewSource = '';
    $('apply-migration').disabled = true;
  };
  $('migration-form').onsubmit = action(async (event) => {
    event.preventDefault();
    await preview();
  });
  $('apply-migration').onclick = async () => {
    const b = $('apply-migration');
    b.disabled = true;
    try {
      if (!previewSource || previewSource !== $('source').value.trim()) {
        throw new Error('请先重新预览');
      }
      notice('正在迁移…');
      const report = await api('migrate', {
        source: previewSource,
        apply: true,
      });
      $('migration-result').replaceChildren(renderReport(report));
      await migrations();
      notice('补迁移完成，原始数据与偏好备份已保留。');
    } catch (e) {
      notice(e.message, true);
    }
  };
}

export { migrations, invalidatePreview, bindMigration };
