import { useEffect, useState } from 'react';
import { api, token } from '../api';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { EditorSelect, editors } from '../components/editor-select';
import type { PanelActions, Settings } from '../types';

export function SettingsPanel({
  active,
  closed,
  pending,
  act,
  message,
  invalidate,
}: PanelActions & { invalidate: () => void }) {
  const [settings, setSettings] = useState<Settings>({
    roots: [],
    editor: 'vscode',
    editors: {},
    managed: false,
  });
  const [roots, setRoots] = useState('');

  const [loaded, setLoaded] = useState(false);
  useEffect(() => {
    if (!token || closed || !active) return;
    const controller = new AbortController();
    setLoaded(false);
    api<Settings>('settings', undefined, controller.signal)
      .then((data) => {
        if (!controller.signal.aborted) {
          setSettings(data);
          setRoots((data.roots || []).join('\n'));
          setLoaded(true);
        }
      })
      .catch((e) => {
        if (!controller.signal.aborted) message(e.message, true);
      });
    return () => controller.abort();
  }, [active, closed]);
  const disabled = closed || pending !== null || !loaded;

  return (
    <>
      <div className="heading">
        <div>
          <h2>按你的习惯配置</h2>
          <p>保存后下一次 Alfred 查询即生效。Git 根目录变更后，请刷新索引。</p>
        </div>
      </div>
      <form
        id="settings-form"
        className="card form"
        onSubmit={(e) => {
          e.preventDefault();
          if (!loaded) return;
          void act('settings', async () => {
            await api('settings', {
              roots: roots
                .split('\n')
                .map((s) => s.trim())
                .filter(Boolean),
              editor: settings.editor,
              editors: Object.fromEntries(
                Object.keys(editors).map((key) => [key, (settings.editors[key] || '').trim()]),
              ),
            });
            invalidate();
            const data = await api<Settings>('settings');
            setSettings(data);
            setRoots((data.roots || []).join('\n'));
            message(
              '配置已保存，下次 Alfred 查询即生效。修改了 Git 根目录时，请到项目记录刷新索引。',
            );
          });
        }}
      >
        <label htmlFor="roots">Git 扫描根目录</label>
        <p className="hint">
          每行一个绝对路径，支持 ~/。留空则只使用 VS Code 历史；普通子目录需先用 VS Code 打开。
        </p>
        <textarea
          id="roots"
          rows={5}
          placeholder="~/Work/Code"
          value={roots}
          disabled={disabled}
          onChange={(e) => setRoots(e.target.value)}
        />
        <label htmlFor="default-editor">默认 IDE</label>
        <EditorSelect
          id="default-editor"
          value={settings.editor}
          disabled={disabled}
          onChange={(e) => setSettings({ ...settings, editor: e.target.value })}
        />
        <details>
          <summary>IDE CLI 路径（可选）</summary>
          <p className="hint">留空自动查找。填写可执行文件完整路径，不包含参数。</p>
          <div id="editor-paths" className="grid">
            {Object.entries(editors).map(([key, label]) => (
              <div key={key}>
                <label htmlFor={'editor-' + key}>{label}</label>
                <Input
                  id={'editor-' + key}
                  value={settings.editors[key] || ''}
                  disabled={disabled}
                  onChange={(e) =>
                    setSettings({
                      ...settings,
                      editors: { ...settings.editors, [key]: e.target.value },
                    })
                  }
                />
              </div>
            ))}
          </div>
        </details>
        <p id="settings-mode" className="hint">
          {settings.managed
            ? '当前使用面板设置，优先于 Alfred 变量和 config.json 顶层同名项。'
            : '当前使用 Alfred 变量 / config.json；保存后，这里的根目录与 IDE 设置优先生效。'}
        </p>
        <div className="actions">
          <Button type="submit" variant="default" disabled={disabled}>
            保存配置
          </Button>
          <Button
            type="button"
            id="reset-settings"
            disabled={disabled}
            onClick={() =>
              act('reset', async () => {
                await api('settings/reset', {});
                invalidate();
                const data = await api<Settings>('settings');
                setSettings(data);
                setRoots((data.roots || []).join('\n'));
                message('已恢复使用 Alfred 变量和高级配置。');
              })
            }
          >
            恢复使用 Alfred / 高级配置
          </Button>
        </div>
      </form>
    </>
  );
}
