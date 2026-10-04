#!/usr/bin/env python3
"""Generate the reviewable Alfred wiring; Python is a build tool only."""
import json
from pathlib import Path
import plistlib

ROOT = Path(__file__).resolve().parent.parent

def workflow():
    objects, connections, positions = [], {}, {}
    def add(uid, kind, config, version=1):
        objects.append({'uid': uid, 'type': 'alfred.workflow.' + kind, 'version': version, 'config': config})
        positions[uid] = {'xpos': float(100 + len(objects) % 4 * 240), 'ypos': float(100 + len(objects) // 4 * 160)}
    def link(source, destination, modifier=0, subtitle=''):
        connections.setdefault(source, []).append({'destinationuid': destination, 'modifiers': modifier, 'modifiersubtext': subtitle, 'vitoclose': False})
    def script(text):
        return {'script': text, 'scriptargtype': 1, 'scriptfile': '', 'type': 0, 'escaping': 102, 'concurrently': False}
    def filter_config(keyword, text, title):
        return {'keyword': keyword, 'script': text, 'scriptargtype': 1, 'scriptfile': '', 'type': 0,
                'escaping': 127, 'title': title, 'subtext': '搜索目录和远程项目', 'runningsubtext': '搜索中…',
                'argumenttype': 1, 'argumenttrimmode': 0, 'argumenttreatemptyqueryasnil': True,
                'withspace': True, 'alfredfiltersresults': False, 'alfredfiltersresultsmatchmode': 2,
                'queuedelaycustom': 0, 'queuedelayimmediatelyinitially': True, 'queuedelaymode': 0, 'queuemode': 1}
    add('SEARCH', 'input.scriptfilter', filter_config('vsc', './vsc query "$1"', '打开项目'), 3)
    capture = 'message=$(./vsc open --request "$1" 2>&1)\nstatus=$?\nif [ "$status" -ne 0 ]; then printf "%s" "$message"; fi\nexit "$status"'
    add('OPEN', 'action.script', script(capture), 2)
    add('CLEAR', 'utility.argument', {'argument': '', 'passthroughargument': False, 'variables': {}})
    add('EDITORS', 'input.scriptfilter', filter_config('', './vsc editors --target "$vsc_target" --query "$1"', '选择 IDE'), 3)
    add('REMEMBER', 'action.script', script(capture.replace('open --request', 'open --remember --request')), 2)
    add('HIDE', 'action.script', script('./vsc hide --target "$vsc_target"'), 2)
    add('PIN', 'action.script', script('./vsc pin --target "$vsc_target"'), 2)
    add('MANAGE', 'input.listfilter', {'keyword': 'vscli', 'withspace': True, 'argumenttype': 1,
        'argumenttrimmode': 0, 'fixedorder': True, 'matchmode': 0, 'title': '管理项目', 'subtext': '',
        'items': json.dumps([
            {'title': '打开记录管理面板', 'subtitle': '管理隐藏、固定、IDE、Git 根目录与迁移结果', 'arg': 'manage'},
            {'title': '刷新 Git 仓库索引', 'subtitle': 'VS Code 历史自动读取，无需重建', 'arg': 'rebuild'},
            {'title': '恢复所有隐藏项目', 'subtitle': '不修改 VS Code 历史或磁盘文件', 'arg': 'restore-all'},
            {'title': '检查环境', 'subtitle': '输出 IDE、历史来源和索引诊断', 'arg': 'doctor'},
            {'title': '生成高级配置', 'subtitle': '输出配置文件位置，不覆盖已有配置', 'arg': 'config-init'},
        ], ensure_ascii=False)})
    add('MANAGE_ACTION', 'action.script', script('./vsc "$1"'), 2)
    add('NOTICE', 'output.notification', {'title': 'VSC', 'text': '{query}', 'onlyshowifquerypopulated': True, 'lastpathcomponent': False, 'removeextension': False})
    for modifier, destination, text in [(0, 'OPEN', ''), (131072, 'OPEN', '新窗口打开'),
          (1048576, 'CLEAR', '选择 IDE'), (262144, 'HIDE', '隐藏项目'), (524288, 'PIN', '固定 / 取消固定')]:
        link('SEARCH', destination, modifier, text)
    link('CLEAR', 'EDITORS'); link('EDITORS', 'OPEN'); link('EDITORS', 'REMEMBER', 1048576, '记住此项目的 IDE')
    link('MANAGE', 'MANAGE_ACTION')
    for source in ('OPEN', 'REMEMBER', 'HIDE', 'PIN', 'MANAGE_ACTION'):
        link(source, 'NOTICE')
    settings = [('VSC_DIRECTORIES', 'Git 项目根目录', '~/Code', '逗号或换行分隔；在管理面板保存设置后，以面板为准'),
                ('VSC_DEFAULT_EDITOR', '默认 IDE', 'vscode', 'vscode / zed / trae / cursor'),
                ('VSC_IDE_PATH', 'VS Code 数据目录（可选）', '', '默认自动寻找 VS Code 数据；不是应用程序路径')]
    for name in ('VSCODE', 'ZED', 'TRAE', 'CURSOR'):
        settings.append(('VSC_EDITOR_' + name, name + ' CLI（可选）', '', '可执行文件的完整路径；不填写参数或 shell 命令'))
    return {'bundleid': 'com.harvey.alfredapp.vsc', 'name': 'vsc', 'createdby': 'Harvey Zhang',
        'description': 'VS Code 目录与 Git 仓库的快速入口', 'version': json.loads((ROOT/'package.json').read_text())['version'],
        'webaddress': 'https://github.com/HarveyZgit/alfred.vsc', 'disabled': False,
        'readme': '无需安装 Python、Node 或 Go。Enter 打开；Cmd 选择 IDE；Shift 新窗口；Ctrl 隐藏；Alt 固定。远程不查询分支。',
        'objects': objects, 'connections': connections, 'uidata': positions,
        'variables': {name: default for name, _, default, _ in settings},
        'userconfigurationconfig': [{'variable': name, 'label': label, 'description': description, 'type': 'textfield',
            'config': {'default': default, 'placeholder': default, 'required': False, 'trim': True}} for name, label, default, description in settings]}

if __name__ == '__main__':
    (ROOT/'public/info.plist').write_bytes(plistlib.dumps(workflow(), sort_keys=False))
