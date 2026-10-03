#!/usr/bin/env python3
"""Verify universal binary slices, archive modes and the Alfred action graph."""
import argparse
import json
from pathlib import Path
import plistlib
import struct
import zipfile

ROOT=Path(__file__).resolve().parent.parent

def verify(package):
    with zipfile.ZipFile(package) as archive:
        assert archive.testzip() is None
        expected={'vsc','info.plist','icon.png','assets/folder.png','assets/remote.png','README.md','LICENSE','THIRD_PARTY_NOTICES.txt'}
        expected.update(f'docs/{name}.zh-CN.md' for name in ('ARCHITECTURE','VALIDATION','MACOS-ACCEPTANCE'))
        assert set(archive.namelist())==expected, archive.namelist()
        assert archive.getinfo('vsc').external_attr >> 16 & 0o777 == 0o755
        binary=archive.read('vsc')
        magic,count=struct.unpack_from('>II',binary)
        assert (magic,count)==(0xcafebabe,2)
        architectures={0x0100000c:'arm64',0x01000007:'amd64'}
        seen=set()
        for i in range(count):
            cpu,subtype,offset,size,align=struct.unpack_from('>IIIII',binary,8+20*i)
            assert offset % (1<<align)==0 and offset+size<=len(binary)
            assert cpu not in seen; seen.add(cpu)
            thin=(ROOT/f'build/vsc-darwin-{architectures[cpu]}').read_bytes()
            assert binary[offset:offset+size]==thin
        assert seen==set(architectures)
        config=plistlib.loads(archive.read('info.plist'))
        assert config['version']==json.loads((ROOT/'package.json').read_text())['version']
        objects={o['uid']:o for o in config['objects']}
        assert len(objects)==len(config['objects'])
        for source,connections in config['connections'].items():
            assert source in objects
            for edge in connections: assert edge['destinationuid'] in objects
        routes={e['modifiers']:e['destinationuid'] for e in config['connections']['SEARCH']}
        assert routes=={0:'OPEN',131072:'OPEN',1048576:'CLEAR',262144:'HIDE',524288:'PIN'}
        assert objects['SEARCH']['config']['script']=='./vsc query "$1"'
        assert '--target "$vsc_target"' in objects['EDITORS']['config']['script']
        for obj in objects.values():
            script=obj['config'].get('script','')
            assert 'python' not in script and 'node ' not in script
        if config['bundleid'].endswith('.dev'):
            assert objects['SEARCH']['config']['keyword']=='vscd'
            assert config['variables']['VSC_DEV_MODE']=='1'
        print(f'PASS {package}: universal arm64/amd64, executable mode, CRC, action graph, native scripts')

if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('packages',nargs='*',type=Path)
    args=parser.parse_args()
    for package in args.packages or [ROOT/'build/vsc.alfredworkflow']: verify(package)
