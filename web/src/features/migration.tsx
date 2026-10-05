import { useEffect, useRef, useState } from 'react';
import { api, token } from '../api';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import type { PanelActions, Report, History } from '../types';

function MigrationReport({ report, historical = false }: { report: Report; historical?: boolean }) {
  return (
    <div className="card report">
      <p className="hint">
        {historical
          ? '首次迁移时的快照（可能过期，请重新预览）'
          : report.applied
            ? '最近一次执行结果（当前范围请重新预览）'
            : '当前迁移预览'}
      </p>
      <p>
        范围内目录 {report.directory_order_imported} / {report.records} · 范围内隐藏项{' '}
        {report.hidden_imported} / {report.hidden_records} · 未迁移{' '}
        {(report.unmigrated || []).length}
      </p>
      <p className="path">来源：{report.source}</p>
      {report.applied && (
        <p className="hint">
          {report.already_applied ? '这些记录已处理，无需重复写入。' : '迁移已完成。'}
        </p>
      )}
      {report.notes?.length ? (
        <p className="hint whitespace-pre-line">{report.notes.join('\n')}</p>
      ) : null}
      {report.unmigrated?.map((item, i) => (
        <div className="migration-item" key={i}>
          <p>{item.original}</p>
          <p className="hint">{item.how_to_migrate}</p>
        </div>
      ))}
    </div>
  );
}

export function Migration({
  active,
  closed,
  pending,
  act,
  message,
  settingsRevision,
}: PanelActions & { settingsRevision: number }) {
  const [source, setSource] = useState('');
  const sourceRef = useRef('');
  const [previewSource, setPreviewSource] = useState<string | null>(null);
  const [report, setReport] = useState<Report | null>(null);
  const [history, setHistory] = useState<History[]>([]);

  const invalidate = () => {
    setPreviewSource(null);
    setReport(null);
  };
  useEffect(invalidate, [settingsRevision]);
  useEffect(() => {
    if (!token || closed || !active) return;
    const controller = new AbortController();
    api<History[]>('migrations', undefined, controller.signal)
      .then((data) => {
        if (!controller.signal.aborted) setHistory(data);
      })
      .catch((e) => {
        if (!controller.signal.aborted) message(e.message, true);
      });
    return () => controller.abort();
  }, [active, closed]);
  async function preview(value = source) {
    invalidate();
    message('正在按当前来源检查迁移范围…');
    const normalized = value.trim();
    const result = await api<Report>('migrate', { source: normalized, apply: false });
    if (sourceRef.current.trim() !== normalized) return;
    setReport(result);
    setPreviewSource(normalized);
    message(
      result.already_applied
        ? '当前范围已迁移，无需重复执行。'
        : '预览完成。确认列表后可执行补迁移。',
    );
  }

  const disabled = closed || pending !== null;
  return (
    <>
      <div className="heading">
        <div>
          <h2>接续旧版记录</h2>
          <p>迁移顺序和隐藏偏好。仅接纳当前来源内的目录，原始备份会保留。</p>
        </div>
      </div>
      <form
        id="migration-form"
        className="card form"
        onSubmit={(e) => {
          e.preventDefault();
          void act('preview', () => preview());
        }}
      >
        <label htmlFor="source">旧版数据目录或 .records.cache.json 路径</label>
        <Input
          id="source"
          placeholder="/Users/…/vsc-old-data…/.records.cache.json"
          required
          value={source}
          disabled={disabled}
          onChange={(e) => {
            setSource(e.target.value);
            sourceRef.current = e.target.value;
            invalidate();
          }}
        />
        <div className="actions">
          <Button type="submit" disabled={disabled}>
            预览当前迁移范围
          </Button>
          <Button
            type="button"
            id="apply-migration"
            variant="default"
            disabled={disabled || !previewSource || report?.already_applied || report?.applied}
            onClick={() =>
              act('apply', async () => {
                if (!previewSource || previewSource !== source.trim())
                  throw new Error('请先重新预览');
                setPreviewSource(null);
                message('正在迁移…');
                setReport(await api<Report>('migrate', { source: previewSource, apply: true }));
                setHistory(await api<History[]>('migrations'));
                message('补迁移完成，原始数据与偏好备份已保留。');
              })
            }
          >
            执行补迁移
          </Button>
        </div>
        <p className="hint">
          先用 VS Code 打开未纳入的目录，或在「目录与 IDE」中加入 Git
          根目录，再预览并补迁移。文件不迁移。
        </p>
      </form>
      <div id="migration-result">{report && <MigrationReport report={report} />}</div>
      <h3 className="history-title">已保存的迁移结果</h3>
      <div id="migration-history">
        {history.map((item, i) => (
          <div key={i}>
            <MigrationReport {...item} />
            <Button
              disabled={disabled}
              onClick={() =>
                act('preview', async () => {
                  setSource(item.report.source);
                  sourceRef.current = item.report.source;
                  await preview(item.report.source);
                })
              }
            >
              使用此来源重新预览
            </Button>
          </div>
        ))}
        {!history.length && <p className="hint">还没有迁移记录。</p>}
      </div>
    </>
  );
}
