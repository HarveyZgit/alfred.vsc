// Optional UI acceptance: requires Playwright and Chromium on the developer machine.
// No frontend dependencies are shipped in the workflow.
const { chromium } = require('playwright');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawn, execFileSync } = require('node:child_process');
const { once } = require('node:events');
const readline = require('node:readline');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
(async () => {
  const binary = path.resolve(process.argv[2] || 'build/vsc-linux-amd64');
  const base = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), 'vsc-panel-browser-')));
  const repo = path.join(base, 'projects', 'Atlas');
  fs.mkdirSync(path.join(repo, '.git'), { recursive: true });
  fs.writeFileSync(path.join(repo, '.git/HEAD'), 'ref: refs/heads/main\n');
  const source = path.join(base, 'legacy.json');
  fs.writeFileSync(
    source,
    JSON.stringify({
      records: [
        { path: repo, type: 'folder' },
        { path: path.join(base, 'outside'), type: 'folder' },
      ],
      trash: {},
    }),
  );
  execFileSync('python3', [
    '-c',
    `import sqlite3,json,sys,pathlib
p=pathlib.Path(sys.argv[1]); db=sqlite3.connect(p/'state.vscdb'); db.execute('CREATE TABLE ItemTable (key TEXT PRIMARY KEY,value TEXT)')
entries=[{'folderUri':(p/'projects/Atlas').as_uri()},{'folderUri':'vscode-remote://ssh-remote+devbox/srv/API'},{'folderUri':(p/'Archived').as_uri()}]
for i in range(55): entries.append({'folderUri':(p/('Demo-%02d'%i)).as_uri()})
db.execute('INSERT INTO ItemTable VALUES (?,?)',('recently.opened',json.dumps({'entries':entries}))); db.commit(); db.close()
`,
    base,
  ]);
  const env = Object.fromEntries(
    Object.entries(process.env).filter(([k]) => !k.startsWith('VSC_') && !k.startsWith('alfred_')),
  );
  Object.assign(env, {
    VSC_DATA_DIR: path.join(base, 'data'),
    VSC_DB_PATH: path.join(base, 'state.vscdb'),
    VSC_DIRECTORIES: path.join(base, 'projects'),
    VSC_NO_BACKGROUND: '1',
  });
  const server = spawn(binary, ['manage', '--serve'], { env, stdio: ['ignore', 'pipe', 'pipe'] });
  const errors = [];
  let browser;
  const startupTimeout = setTimeout(() => server.kill(), 5000);
  try {
    const lines = readline.createInterface({ input: server.stdout });
    const [url] = await once(lines, 'line');
    clearTimeout(startupTimeout);
    browser = await chromium.launch({
      executablePath: process.env.CHROMIUM_PATH || '/usr/bin/chromium',
      headless: true,
      args: ['--no-sandbox'],
    });
    const page = await browser.newPage({ viewport: { width: 1280, height: 960 } });
    page.on('pageerror', (e) => errors.push(e.message));
    await page.goto(url);
    await page.getByText('1 / 2 页 · 58 项', { exact: true }).waitFor();
    await page.getByRole('button', { name: '下一页', exact: true }).click();
    await page.getByText('2 / 2 页 · 58 项', { exact: true }).waitFor();
    assert(await page.getByRole('button', { name: '下一页', exact: true }).isDisabled());
    await page.getByRole('searchbox').fill('Atlas');
    await page.getByText('1 / 1 页 · 1 项', { exact: true }).waitFor();
    await page.getByRole('button', { name: '隐藏', exact: true }).click();
    await page.getByRole('button', { name: '恢复显示', exact: true }).waitFor();
    await page.getByLabel('筛选记录').selectOption('hidden');
    await page.getByRole('button', { name: '恢复显示', exact: true }).click();
    await page
      .getByText('没有匹配的项目。可修改筛选条件或刷新 Git 索引。', { exact: true })
      .waitFor();
    await page.getByLabel('筛选记录').selectOption('');
    await page.getByRole('button', { name: '固定', exact: true }).click();
    await page.getByRole('button', { name: '取消固定', exact: true }).waitFor();
    await page.getByLabel('Atlas 的 IDE').selectOption('zed');
    await page.reload();
    await page.getByRole('searchbox').fill('Atlas');
    await page.getByText('1 / 1 页 · 1 项', { exact: true }).waitFor();
    assert.equal(await page.getByLabel('Atlas 的 IDE').inputValue(), 'zed');
    await page.getByRole('button', { name: '目录与 IDE', exact: true }).click();
    await page.getByLabel('Git 扫描根目录').fill(path.join(base, 'projects'));
    await page.getByLabel('默认 IDE', { exact: true }).selectOption('cursor');
    await page.getByRole('button', { name: '保存配置', exact: true }).click();
    await page
      .getByText('当前使用面板设置，优先于 Alfred 变量和 config.json 顶层同名项。', { exact: true })
      .waitFor();
    await page.getByRole('button', { name: '旧版迁移', exact: true }).click();
    await page.getByLabel('旧版数据目录或 .records.cache.json 路径').fill(source);
    await page.getByRole('button', { name: '预览当前迁移范围', exact: true }).click();
    await page.getByText('预览完成。确认列表后可执行补迁移。', { exact: true }).waitFor();
    await page.getByRole('button', { name: '执行补迁移', exact: true }).click();
    await page.getByText('补迁移完成，原始数据与偏好备份已保留。', { exact: true }).waitFor();
    await page.getByRole('button', { name: '项目记录', exact: true }).click();
    await page.getByRole('searchbox').fill('');
    await page.getByText('1 / 2 页 · 58 项', { exact: true }).waitFor();
    await page.getByRole('button', { name: '刷新 Git 索引', exact: true }).click();
    await page.getByText('Git 索引已刷新。', { exact: true }).waitFor();
    fs.mkdirSync('build', { recursive: true });
    await page.screenshot({ path: 'build/panel-browser.png' });
    await page.setViewportSize({ width: 390, height: 844 });
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    await page.getByRole('button', { name: '关闭面板服务', exact: true }).click();
    await page.getByText('面板服务已关闭，可以关闭此标签页。', { exact: true }).waitFor();
    assert.deepEqual(errors, []);
    fs.writeFileSync(
      'build/panel-browser-acceptance.json',
      JSON.stringify(
        {
          binary_sha256: crypto.createHash('sha256').update(fs.readFileSync(binary)).digest('hex'),
          browser: await browser.version(),
          checks: [
            'pagination, debounced search, hide/restore, pin, IDE persistence',
            'browser reload keeps session',
            'save settings, migration preview/apply and saved result',
            'embedded CSS/JS, responsive width, no page errors, close service',
          ],
        },
        null,
        2,
      ) + '\n',
    );
    console.log(
      'PASS: Chromium panel interactions, reload, settings, migration, responsive layout; screenshot and report in build/',
    );
  } finally {
    clearTimeout(startupTimeout);
    if (browser) await browser.close();
    if (server.exitCode === null) {
      const exited = once(server, 'exit');
      server.kill();
      await exited;
    }
    fs.rmSync(base, { recursive: true, force: true });
  }
})().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
