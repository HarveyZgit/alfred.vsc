'use strict';
const $ = id => document.getElementById(id);
const token = location.hash.slice(1) || sessionStorage.getItem('vsc-session') || '';
if (token) sessionStorage.setItem('vsc-session', token);
history.replaceState(null, '', '/');
const editors = {vscode: 'VS Code', zed: 'Zed', trae: 'Trae', cursor: 'Cursor'};
let page = 0, pages = 1, serial = 0, previewSource = '', timer;
function element(tag, text, className) { const e = document.createElement(tag); if (text !== undefined) e.textContent = text; if (className) e.className = className; return e; }
function notice(text, error = false) { $('notice').textContent = text; $('notice').className = error ? 'error' : ''; }
async function api(path, body) {
  const response = await fetch('/api/' + path, {method: body === undefined ? 'GET' : 'POST', headers: {Authorization: 'Bearer ' + token, 'Content-Type': 'application/json'}, body: body === undefined ? undefined : JSON.stringify(body)});
  const text = await response.text();
  let data; try { data = JSON.parse(text); } catch { throw new Error(text || '面板服务已退出，请从 Alfred 重新打开。'); }
  if (!response.ok) throw new Error(data.error || text);
  return data;
}
function action(fn) { return async event => { const button = event?.currentTarget; if (button?.tagName === 'BUTTON') button.disabled = true; try { await fn(event); } catch(e) { notice(e.message, true); } finally { if (button?.tagName === 'BUTTON') button.disabled = false; } }; }
function button(text, fn) { const b = element('button', text); b.type = 'button'; b.addEventListener('click', action(fn)); return b; }
function editorSelect(value, inherit = false) { const s = element('select'); if (inherit) { const o = element('option', '跟随默认 IDE'); o.value = ''; s.append(o); } for (const [key, label] of Object.entries(editors)) { const o = element('option', label); o.value = key; s.append(o); } s.value = value; return s; }
async function records() {
  const request = ++serial;
  const data = await api('records?' + new URLSearchParams({q:$('search').value, filter:$('filter').value, page}));
  if (request !== serial) return;
  page = data.page; pages = data.pages;
  $('metrics').replaceChildren(...[[data.all,'项目'],[data.hidden,'已隐藏'],[data.pinned,'已固定']].map(([n,t]) => { const e = element('div',undefined,'metric'); e.append(element('strong',String(n)),document.createTextNode(t)); return e; }));
  $('warnings').textContent = [...(data.warnings || []), ...(data.index_stale ? ['Git 索引尚未建立或已过期，可点击「刷新 Git 索引」。'] : [])].join(' · ');
  $('list').replaceChildren();
  for (const row of data.items) {
    const card = element('article', undefined, 'record');
    const main = element('div', undefined, 'record-main'); main.append(element('h3',row.name),element('div',row.kind === 'remote' ? row.uri : row.path,'path'));
    const badges = element('div',undefined,'badges');
    badges.append(element('span',row.status,'badge' + (!['可用','远程 · 未探测'].includes(row.status) ? ' warn' : '')));
    if(row.branch) badges.append(element('span',row.branch,'badge green'));
    if(row.hidden) badges.append(element('span','已隐藏','badge'));
    if(row.pinned) badges.append(element('span','已固定','badge'));
    main.append(badges);
    const actions = element('div',undefined,'record-actions');
    const update = async body => { await api('record',{id:row.id,...body}); await records(); };
    const select = editorSelect(row.editor,true); select.setAttribute('aria-label',row.name+' 的 IDE'); select.addEventListener('change',action(async()=>{await update({action:'editor',editor:select.value});}));
    actions.append(select,button(row.pinned?'取消固定':'固定',()=>update({action:'pinned',enabled:!row.pinned})),button(row.hidden?'恢复显示':'隐藏',()=>update({action:'hidden',enabled:!row.hidden})));
    card.append(main,actions); $('list').append(card);
  }
  if(!data.items.length) $('list').append(element('div','没有匹配的项目。可修改筛选条件或刷新 Git 索引。','empty'));
  $('page').textContent = `${page+1} / ${pages} 页 · ${data.total} 项`;
  $('prev').disabled = page === 0; $('next').disabled = page + 1 >= pages;
  $('data-dir').textContent = '当前数据目录：'+data.data_dir;
}
async function settings() {
  const data = await api('settings'); $('roots').value = (data.roots || []).join('\n'); $('default-editor').value = data.editor;
  $('editor-paths').replaceChildren();
  for(const [key,label] of Object.entries(editors)){ const wrapper=element('div'); const l=element('label',label); l.htmlFor='editor-'+key; const input=element('input'); input.id='editor-'+key; input.value=data.editors[key] || ''; wrapper.append(l,input); $('editor-paths').append(wrapper); }
  $('settings-mode').textContent = data.managed ? '当前使用面板设置，优先于 Alfred 变量和 config.json 顶层同名项。' : '当前使用 Alfred 变量 / config.json；保存后，这里的根目录与 IDE 设置优先生效。';
}
function renderReport(report, historical = false) {
  const box = element('div',undefined,'card');
  box.append(element('p',historical?'首次迁移时的快照（可能过期，请重新预览）':report.applied?'最近一次执行结果（当前范围请重新预览）':'当前迁移预览','hint'));
  box.append(element('p',`范围内目录 ${report.directory_order_imported} / ${report.records} · 范围内隐藏项 ${report.hidden_imported} / ${report.hidden_records} · 未迁移 ${(report.unmigrated || []).length}`,'report-summary'));
  box.append(element('p','来源：'+report.source,'path'));
  if(report.applied) box.append(element('p',report.already_applied?'这些记录已处理，无需重复写入。':'迁移已完成。','hint'));
  if(report.notes?.length) box.append(element('p',report.notes.join('\n'),'hint'));
  for(const item of report.unmigrated || []) { const row=element('div',undefined,'migration-item'); row.append(element('p',item.original),element('p',item.how_to_migrate,'hint')); box.append(row); }
  return box;
}
async function migrations() {
  const data = await api('migrations'); $('migration-history').replaceChildren();
  for(const item of data){ const card=renderReport(item.report,item.historical); card.append(button('使用此来源重新预览',async()=>{$('source').value=item.report.source; await preview();})); $('migration-history').append(card); }
  if(!data.length) $('migration-history').append(element('p','还没有迁移记录。','hint'));
}
async function preview(){ previewSource=''; $('apply-migration').disabled=true; notice('正在按当前来源检查迁移范围…'); const source=$('source').value.trim(); const report=await api('migrate',{source,apply:false}); $('migration-result').replaceChildren(renderReport(report)); previewSource=source; $('apply-migration').disabled=report.already_applied; notice(report.already_applied?'当前范围已迁移，无需重复执行。':'预览完成。确认列表后可执行补迁移。'); }
for(const tab of document.querySelectorAll('[data-tab]')) tab.addEventListener('click',action(async()=>{ document.querySelectorAll('[data-tab]').forEach(t=>t.classList.toggle('active',t===tab)); for(const name of ['records','settings','migration']) $(name).hidden=name!==tab.dataset.tab; if(tab.dataset.tab==='records') await records(); if(tab.dataset.tab==='settings') await settings(); if(tab.dataset.tab==='migration') await migrations(); }));
$('search').addEventListener('input',()=>{ clearTimeout(timer); ++serial; page=0; timer=setTimeout(()=>records().catch(e=>notice(e.message,true)),180); });
$('filter').addEventListener('change',action(async()=>{page=0;await records();}));
$('reload').onclick=action(records);
// Navigation keeps the disabled state computed by records().
$('prev').onclick=()=>{if(page>0){page--;records().catch(e=>notice(e.message,true));}};
$('next').onclick=()=>{if(page+1<pages){page++;records().catch(e=>notice(e.message,true));}};
$('rebuild').onclick=action(async()=>{ notice('正在扫描 Git 根目录…'); const result=await api('rebuild',{}); await records(); notice('Git 索引已刷新。'+(result.warnings || []).join(' · ')); });
$('settings-form').onsubmit=action(async event=>{ event.preventDefault(); const paths={}; for(const key of Object.keys(editors)) paths[key]=$('editor-'+key).value.trim(); await api('settings',{roots:$('roots').value.split('\n').map(s=>s.trim()).filter(Boolean),editor:$('default-editor').value,editors:paths}); previewSource=''; $('apply-migration').disabled=true; await settings(); notice('配置已保存，下次 Alfred 查询即生效。修改了 Git 根目录时，请到项目记录刷新索引。'); });
$('reset-settings').onclick=action(async()=>{await api('settings/reset',{}); previewSource=''; $('apply-migration').disabled=true; await settings(); notice('已恢复使用 Alfred 变量和高级配置。');});
$('source').oninput=()=>{previewSource='';$('apply-migration').disabled=true;};
$('migration-form').onsubmit=action(async event=>{event.preventDefault();await preview();});
$('apply-migration').onclick=async()=>{ const b=$('apply-migration'); b.disabled=true; try{ if(!previewSource || previewSource!==$('source').value.trim()) throw new Error('请先重新预览'); notice('正在迁移…'); const report=await api('migrate',{source:previewSource,apply:true}); $('migration-result').replaceChildren(renderReport(report)); await migrations(); notice('补迁移完成，原始数据与偏好备份已保留。'); }catch(e){notice(e.message,true);} };
$('close').onclick=action(async()=>{ await api('close',{}); document.querySelectorAll('button,input,textarea,select').forEach(e=>e.disabled=true); notice('面板服务已关闭，可以关闭此标签页。'); });
$('version').textContent='VSC 3';
if(!token) notice('请从 Alfred 的管理入口重新打开此面板。',true); else records().catch(e=>notice(e.message,true));
