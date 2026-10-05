import { useRef, useState } from 'react';
import { api, token } from './api';
import { Button } from './components/ui/button';
import { Tabs, TabsList, TabsTrigger, TabsContent } from './components/ui/tabs';
import { Records } from './features/records';
import { SettingsPanel } from './features/settings';
import { Migration } from './features/migration';
import logo from '../../public/icon.png';

export function App() {
  const [tab, setTab] = useState('records');
  const [notice, setNotice] = useState({
    text: token ? '' : '请从 Alfred 的管理入口重新打开此面板。',
    error: !token,
  });
  const [closed, setClosed] = useState(false);
  const [pending, setPending] = useState<string | null>(null);
  const pendingRef = useRef(false);
  const [settingsRevision, setSettingsRevision] = useState(0);
  const message = (text: string, error = false) => setNotice({ text, error });
  async function act(key: string, fn: () => Promise<void>) {
    if (pendingRef.current || closed) return;
    pendingRef.current = true;
    setPending(key);
    try {
      await fn();
    } catch (e) {
      message(e instanceof Error ? e.message : String(e), true);
    } finally {
      pendingRef.current = false;
      setPending(null);
    }
  }
  const disabled = closed || pending !== null;
  return (
    <>
      <header>
        <div className="brand">
          <img className="logo" src={logo} alt="VSC" width={42} height={42} />
          <div>
            <h1>项目管理</h1>
            <p>VSC · 你的项目入口</p>
          </div>
        </div>
        <Button
          id="close"
          disabled={disabled}
          onClick={() =>
            act('close', async () => {
              await api('close', {});
              setClosed(true);
              message('面板服务已关闭，可以关闭此标签页。');
            })
          }
        >
          关闭面板服务
        </Button>
      </header>
      <main>
        <Tabs value={tab} onValueChange={setTab}>
          <TabsList aria-label="管理页面">
            {[
              ['records', '项目记录'],
              ['settings', '目录与 IDE'],
              ['migration', '旧版迁移'],
            ].map(([value, label]) => (
              <TabsTrigger
                key={value}
                value={value}
                disabled={disabled}
                id={'tab-' + value}
                aria-controls={value}
                data-tab={value}
              >
                {label}
              </TabsTrigger>
            ))}
          </TabsList>
          <div
            id="notice"
            role="status"
            aria-live="polite"
            className={notice.error ? 'notice error' : 'notice'}
          >
            {notice.text}
          </div>
          <TabsContent
            value="records"
            id="records"
            aria-labelledby="tab-records"
            forceMount
            hidden={tab !== 'records'}
          >
            <Records
              active={tab === 'records'}
              closed={closed}
              pending={pending}
              act={act}
              message={message}
            />
          </TabsContent>
          <TabsContent
            value="settings"
            id="settings"
            aria-labelledby="tab-settings"
            forceMount
            hidden={tab !== 'settings'}
          >
            <SettingsPanel
              active={tab === 'settings'}
              closed={closed}
              pending={pending}
              act={act}
              message={message}
              invalidate={() => setSettingsRevision((v) => v + 1)}
            />
          </TabsContent>
          <TabsContent
            value="migration"
            id="migration"
            aria-labelledby="tab-migration"
            forceMount
            hidden={tab !== 'migration'}
          >
            <Migration
              active={tab === 'migration'}
              closed={closed}
              pending={pending}
              act={act}
              message={message}
              settingsRevision={settingsRevision}
            />
          </TabsContent>
        </Tabs>
        <footer>
          仅本机访问 · 20 分钟无操作自动退出 · <span id="version">VSC 3</span>
        </footer>
      </main>
    </>
  );
}
