const token = location.hash.slice(1) || sessionStorage.getItem('vsc-session') || '';
if (token) {
  sessionStorage.setItem('vsc-session', token);
}
history.replaceState(null, '', '/');

async function api(path, body) {
  const response = await fetch('/api/' + path, {
    method: body === undefined ? 'GET' : 'POST',
    headers: {
      Authorization: 'Bearer ' + token,
      'Content-Type': 'application/json',
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await response.text();
  let data;
  try {
    data = JSON.parse(text);
  } catch {
    throw new Error(text || '面板服务已退出，请从 Alfred 重新打开。');
  }
  if (!response.ok) {
    throw new Error(data.error || text);
  }
  return data;
}

export { api, token };
