import { filterRecords } from '.';
import { inputParser } from '../input';

function testTemplate(input: string) {
  return filterRecords(inputParser(input))([
    {
      __vsc_id__: '0ffc8447fcbae99e76b2b617ecb36b7e',
      name: 'tools',
      path: 'file:///Users/example/Code/alfred/tools',
      icon: {
        path: './assets/folder.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: '1edd63687ab30a92b325ce77ec459ca2',
      name: 'vsc',
      path: 'file:///Users/example/Code/alfred/vsc',
      icon: {
        path: './assets/folder.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: '668392b3fa0b759da560800b2edd1f33',
      name: 'temp',
      path: 'file:///Users/example/Code/temp',
      icon: {
        path: './assets/folder.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: 'b1f0b5de1106eca5212c97640c4b3524',
      name: 'sample-fe',
      path: 'vscode-remote://ssh-remote+example-host-a/home/example/workspace/sample-fe',
      icon: {
        path: './assets/remote.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: '4468d062d13611a2475969e25999f348',
      name: 'sample-fe-lib',
      path: 'vscode-remote://ssh-remote+example-host-b/home/example/workspace/sample-fe-lib',
      icon: {
        path: './assets/remote.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: '1915dd2e5c719706d5d3ea46c36dfcf2',
      name: 'user.workflow.FA85466B-F29A-5F5C-A7ED-7B99AAF52325',
      path: 'file:///Users/example/Library/Application Support/Alfred/Alfred.alfredpreferences/workflows/user.workflow.FA85466B-F29A-5F5C-A7ED-7B99AAF52325',
      icon: {
        path: './assets/folder.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: '5f09c5d7f90761c264aab7ae000bddf6',
      name: 'user.workflow.harvey.vsc.development',
      path: 'file:///Users/example/Library/Application Support/Alfred/Alfred.alfredpreferences/workflows/user.workflow.harvey.vsc.development',
      icon: {
        path: './assets/folder.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: '30035a9de2fb337a74e04af0f6e399c6',
      name: 'docker',
      path: 'vscode-remote://ssh-remote+192.0.2.10/root/docker',
      icon: {
        path: './assets/remote.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: 'eb0b4425a12f674a1a7d5d8f625dd34e',
      name: 'user.workflow.34879A5F-AD44-53D0-B834-A5BAC5FF0551',
      path: 'file:///Users/example/Library/Application Support/Alfred/Alfred.alfredpreferences/workflows/user.workflow.34879A5F-AD44-53D0-B834-A5BAC5FF0551',
      icon: {
        path: './assets/folder.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: 'ce79407860692840c18831045ce3f401',
      name: '.records.cache.json',
      path: 'file:///Users/example/Library/Application Support/Alfred/Alfred.alfredpreferences/workflows/user.workflow.harvey.vsc.development/.records.cache.json',
      icon: {
        path: './assets/file.png',
      },
      extra: {
        from: 'getRecordsFromVscodeDB',
      },
    },
    {
      __vsc_id__: '80630053775f7af2e2cc2fb34c1fa241',
      name: 'Application',
      path: 'file:///Users/example/Library/Application',
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
