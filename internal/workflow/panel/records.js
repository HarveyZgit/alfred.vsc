import { api } from './api.js';
import { $, element, notice, action, button, editorSelect } from './dom.js';

let page = 0;
let pages = 1;
let serial = 0;
let timer;

async function records() {
  const request = ++serial;
  const data = await api(
    'records?' +
      new URLSearchParams({
        q: $('search').value,
        filter: $('filter').value,
        page,
      }),
  );
  if (request !== serial) {
    return;
  }
  page = data.page;
  pages = data.pages;
  $('metrics').replaceChildren(
    ...[
      [data.all, '项目'],
      [data.hidden, '已隐藏'],
      [data.pinned, '已固定'],
    ].map(([n, t]) => {
      const e = element('div', undefined, 'metric');
      e.append(element('strong', String(n)), document.createTextNode(t));
      return e;
    }),
  );
  $('warnings').textContent = [
    ...(data.warnings || []),
    ...(data.index_stale ? ['Git 索引尚未建立或已过期，可点击「刷新 Git 索引」。'] : []),
  ].join(' · ');
  $('list').replaceChildren();
  for (const row of data.items) {
    const card = element('article', undefined, 'record');
    const main = element('div', undefined, 'record-main');
    main.append(
      element('h3', row.name),
      element('div', row.kind === 'remote' ? row.uri : row.path, 'path'),
    );
    const badges = element('div', undefined, 'badges');
    badges.append(
      element(
        'span',
        row.status,
        'badge' + (!['可用', '远程 · 未探测'].includes(row.status) ? ' warn' : ''),
      ),
    );
    if (row.branch) {
      badges.append(element('span', row.branch, 'badge green'));
    }
    if (row.hidden) {
      badges.append(element('span', '已隐藏', 'badge'));
    }
    if (row.pinned) {
      badges.append(element('span', '已固定', 'badge'));
    }
    main.append(badges);
    const actions = element('div', undefined, 'record-actions');
    const update = async (body) => {
      await api('record', {
        id: row.id,
        ...body,
      });
      await records();
    };
    const select = editorSelect(row.editor, true);
    select.setAttribute('aria-label', row.name + ' 的 IDE');
    select.addEventListener(
      'change',
      action(async () => {
        await update({
          action: 'editor',
          editor: select.value,
        });
      }),
    );
    actions.append(
      select,
      button(row.pinned ? '取消固定' : '固定', () =>
        update({
          action: 'pinned',
          enabled: !row.pinned,
        }),
      ),
      button(row.hidden ? '恢复显示' : '隐藏', () =>
        update({
          action: 'hidden',
          enabled: !row.hidden,
        }),
      ),
    );
    card.append(main, actions);
    $('list').append(card);
  }
  if (!data.items.length) {
    $('list').append(element('div', '没有匹配的项目。可修改筛选条件或刷新 Git 索引。', 'empty'));
  }
  $('page').textContent = `${page + 1} / ${pages} 页 · ${data.total} 项`;
  $('prev').disabled = page === 0;
  $('next').disabled = page + 1 >= pages;
  $('data-dir').textContent = '当前数据目录：' + data.data_dir;
}

function bindRecords() {
  $('search').addEventListener('input', () => {
    clearTimeout(timer);
    ++serial;
    page = 0;
    timer = setTimeout(() => records().catch((e) => notice(e.message, true)), 180);
  });
  $('filter').addEventListener(
    'change',
    action(async () => {
      page = 0;
      await records();
    }),
  );
  $('reload').onclick = action(records);
  // Navigation keeps the disabled state computed by records().
  $('prev').onclick = () => {
    if (page > 0) {
      page--;
      records().catch((e) => notice(e.message, true));
    }
  };
  $('next').onclick = () => {
    if (page + 1 < pages) {
      page++;
      records().catch((e) => notice(e.message, true));
    }
  };
  $('rebuild').onclick = action(async () => {
    notice('正在扫描 Git 根目录…');
    const result = await api('rebuild', {});
    await records();
    notice('Git 索引已刷新。' + (result.warnings || []).join(' · '));
  });
}

export { records, bindRecords };
