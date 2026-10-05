const $ = (id) => document.getElementById(id);
const editors = {
  vscode: 'VS Code',
  zed: 'Zed',
  trae: 'Trae',
  cursor: 'Cursor',
};

function element(tag, text, className) {
  const e = document.createElement(tag);
  if (text !== undefined) {
    e.textContent = text;
  }
  if (className) {
    e.className = className;
  }
  return e;
}

function notice(text, error = false) {
  $('notice').textContent = text;
  $('notice').className = error ? 'error' : '';
}

function action(fn) {
  return async (event) => {
    const button = event?.currentTarget;
    if (button?.tagName === 'BUTTON') {
      button.disabled = true;
    }
    try {
      await fn(event);
    } catch (e) {
      notice(e.message, true);
    } finally {
      if (button?.tagName === 'BUTTON') {
        button.disabled = false;
      }
    }
  };
}

function button(text, fn) {
  const b = element('button', text);
  b.type = 'button';
  b.addEventListener('click', action(fn));
  return b;
}

function editorSelect(value, inherit = false) {
  const s = element('select');
  if (inherit) {
    const o = element('option', '跟随默认 IDE');
    o.value = '';
    s.append(o);
  }
  for (const [key, label] of Object.entries(editors)) {
    const o = element('option', label);
    o.value = key;
    s.append(o);
  }
  s.value = value;
  return s;
}

export { $, editors, element, notice, action, button, editorSelect };
