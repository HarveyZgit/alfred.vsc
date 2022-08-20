import { filterRecords } from '.';
import { inputParser } from '../input';

function testTemplate(input: string) {
  return filterRecords(inputParser(input))([
    {
      __vsc_id__: '30df9ba4e8c974025c444d588c14329a',
      name: 'tools',
      path: 'file:///Users/harvey/Code/alfred/tools',
      icon: {
        path: './assets/folder.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: '92e8e6703d250e775fc7a3557aa15c9e',
      name: 'vsc',
      path: 'file:///Users/harvey/Code/alfred/vsc',
      icon: {
        path: './assets/folder.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: 'cb3bb8e83013547a29e7ebe13d86d660',
      name: 'temp',
      path: 'file:///Users/harvey/Code/temp',
      icon: {
        path: './assets/folder.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: '8f97572eb1b00d0367f02d531ff7281d',
      name: 'corehr-fe',
      path: 'vscode-remote://ssh-remote+byteide_96572/home/byteide/workspace/corehr-fe',
      icon: {
        path: './assets/remote.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: '8e0e36b3cac88d70ea3e4a2b1dd75474',
      name: 'corehr-fe-lib',
      path: 'vscode-remote://ssh-remote+byteide_97098/home/byteide/workspace/corehr-fe-lib',
      icon: {
        path: './assets/remote.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: '452d5aff18be9932c5d844676742d20e',
      name: 'user.workflow.0354FFD2-0FA5-4BEA-AE87-769C854D5E4C',
      path: 'file:///Users/harvey/Library/Application Support/Alfred/Alfred.alfredpreferences/workflows/user.workflow.0354FFD2-0FA5-4BEA-AE87-769C854D5E4C',
      icon: {
        path: './assets/folder.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: 'c54b0c29864a819146c6c30888a35b0b',
      name: 'user.workflow.harvey.vsc.development',
      path: 'file:///Users/harvey/Library/Application Support/Alfred/Alfred.alfredpreferences/workflows/user.workflow.harvey.vsc.development',
      icon: {
        path: './assets/folder.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: '468897ef3bf706e2ded4dde5b70d5b43',
      name: 'docker',
      path: 'vscode-remote://ssh-remote+82.157.182.50/root/docker',
      icon: {
        path: './assets/remote.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: '2a4060f1ea80de68b187adcaf9c20386',
      name: 'user.workflow.5C90F821-1501-4521-8A2C-39D38673E7F9',
      path: 'file:///Users/harvey/Library/Application Support/Alfred/Alfred.alfredpreferences/workflows/user.workflow.5C90F821-1501-4521-8A2C-39D38673E7F9',
      icon: {
        path: './assets/folder.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: 'fd440b0ad5a34bf7eb06360fda8b7ddc',
      name: '.records.cache.json',
      path: 'file:///Users/harvey/Library/Application Support/Alfred/Alfred.alfredpreferences/workflows/user.workflow.harvey.vsc.development/.records.cache.json',
      icon: {
        path: './assets/file.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: '96f6e919af42dc54dacb2b26e7ed0d09',
      name: 'Application',
      path: 'file:///Users/harvey/Library/Application',
      icon: {
        path: './assets/file.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
  ]);
}

console.log(testTemplate('dir vsc'));
console.log(testTemplate('dirvsc'));
console.log(testTemplate('dir'));
