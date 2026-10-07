import { useEffect, useState } from 'react';
import { api, token } from '../api';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { EditorSelect } from '../components/editor-select';
import type { PanelActions, RecordsData } from '../types';

export function Records({ active, closed, pending, act, message }: PanelActions) {
  const [search, setSearch] = useState('');
  const [query, setQuery] = useState('');
  const [filter, setFilter] = useState('');
  const [page, setPage] = useState(0);
  const [revision, setRevision] = useState(0);
  const [records, setRecords] = useState<RecordsData | null>(null);
  const [loading, setLoading] = useState(false);

  const refresh = () => setRevision((v) => v + 1);
  useEffect(() => {
    const timer = setTimeout(() => {
      setQuery(search);
      setPage(0);
    }, 180);
    return () => clearTimeout(timer);
  }, [search]);
  useEffect(() => {
    if (!token || closed || !active || search !== query) return;
    const controller = new AbortController();
    setLoading(true);
    api<RecordsData>(
      'records?' + new URLSearchParams({ q: query, filter, page: String(page) }),
      undefined,
      controller.signal,
    )
      .then((data) => {
        if (!controller.signal.aborted) {
          setRecords(data);
          if (data.page !== page) setPage(data.page);
        }
      })
      .catch((e) => {
        if (!controller.signal.aborted) message(e.message, true);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [active, search, query, filter, page, revision, closed]);

  const disabled = closed || pending !== null;
  return (
    <>
      <div className="heading">
        <div>
          <h2>所有项目，尽在这里</h2>
          <p>来自 VS Code 打开的目录与配置目录下的 Git 仓库。隐藏的项目可随时恢复。</p>
        </div>
        <Button
          id="rebuild"
          variant="default"
          disabled={disabled}
          onClick={() =>
            act('rebuild', async () => {
              message('正在扫描 Git 根目录…');
              const result = await api<{ warnings?: string[] }>('rebuild', {});
              refresh();
              message('Git 索引已刷新。' + (result.warnings || []).join(' · '));
            })
          }
        >
          刷新 Git 索引
        </Button>
      </div>
      <div id="metrics" className="metrics">
        {[
          [records?.all, '项目'],
          [records?.hidden, '已隐藏'],
          [records?.pinned, '已固定'],
        ].map(([count, label]) => (
          <div className="card metric" key={label}>
            <span>{label}</span>
            <strong>{count ?? '—'}</strong>
          </div>
        ))}
      </div>
      <div className="toolbar">
        <Input
          type="search"
          id="search"
          aria-label="搜索记录"
          placeholder="搜索项目名、路径或远程主机"
          value={search}
          disabled={closed}
          onChange={(e) => setSearch(e.target.value)}
        />
        <select
          id="filter"
          aria-label="筛选记录"
          disabled={closed}
          value={filter}
          onChange={(e) => {
            setFilter(e.target.value);
            setPage(0);
          }}
        >
          {[
            ['', '全部记录'],
            ['hidden', '已隐藏'],
            ['pinned', '已固定'],
            ['local', '本地目录'],
            ['remote', '远程目录'],
          ].map(([key, label]) => (
            <option key={key} value={key}>
              {label}
            </option>
          ))}
        </select>
        <Button id="reload" disabled={disabled || loading} onClick={refresh}>
          刷新列表
        </Button>
      </div>
      <p id="warnings" className="hint">
        {[
          ...(records?.warnings || []),
          ...(records?.index_stale ? ['Git 索引尚未建立或已过期，可点击「刷新 Git 索引」。'] : []),
        ].join(' · ')}
      </p>
      <div id="list" aria-live="polite" aria-busy={loading}>
        {records?.items.map((row) => (
          <article className="card record" key={row.id}>
            <div className="record-main">
              <h3>{row.name}</h3>
              <p className="path">{row.kind === 'remote' ? row.uri : row.path}</p>
              <div className="badges">
                {[row.status, row.branch, row.hidden ? '已隐藏' : '', row.pinned ? '已固定' : '']
                  .filter(Boolean)
                  .map((badge, i) => (
                    <span className="badge" key={i}>
                      {badge}
                    </span>
                  ))}
              </div>
            </div>
            <div className="record-actions">
              <EditorSelect
                inherit
                value={row.editor}
                aria-label={row.name + ' 的 IDE'}
                disabled={disabled}
                onChange={(e) => {
                  const editor = e.target.value;
                  void act(row.id, async () => {
                    await api('record', { id: row.id, action: 'editor', editor });
                    refresh();
                  });
                }}
              />
              {[
                ['pinned', row.pinned ? '取消固定' : '固定', !row.pinned],
                ['hidden', row.hidden ? '恢复显示' : '隐藏', !row.hidden],
              ].map(([action, label, enabled]) => (
                <Button
                  key={String(action)}
                  size="sm"
                  disabled={disabled}
                  onClick={() =>
                    act(row.id, async () => {
                      await api('record', { id: row.id, action, enabled });
                      refresh();
                    })
                  }
                >
                  {label}
                </Button>
              ))}
            </div>
          </article>
        ))}
        {records?.items.length === 0 && (
          <div className="card empty">没有匹配的项目。可修改筛选条件或刷新 Git 索引。</div>
        )}
      </div>
      <div className="pagination">
        <Button
          id="prev"
          disabled={disabled || loading || page === 0}
          onClick={() => setPage((v) => v - 1)}
        >
          上一页
        </Button>
        <span id="page">
          {records
            ? `${records.page + 1} / ${records.pages} 页 · ${records.total} 项`
            : '正在加载…'}
        </span>
        <Button
          id="next"
          disabled={disabled || loading || !records || page + 1 >= records.pages}
          onClick={() => setPage((v) => v + 1)}
        >
          下一页
        </Button>
      </div>
      <p id="data-dir" className="hint">
        {records ? '当前数据目录：' + records.data_dir : ''}
      </p>
    </>
  );
}
