# VSC

An alfred workflow for finds recent VScode open records

# Quick Start

1. [Download](https://github.com/HarveyZgit/alfred.vsc/releases) the latest Workflow
2. Double-click to install the workflow
3. Open your Alfred and input vsc to search

# Features

## Filter by path type

Type the path type to get search result.

For example, `d` in the following image is one of path types, meaning search for folders whose names contain `vsc`.
![Filter by path type](https://raw.githubusercontent.com/HarveyZgit/drawing-bed/master/img/20220820175808.png)

### All path types

| path type  | aliases                          |
| ---------- | -------------------------------- |
| folder     | `folder`, `fo`, `ff`, `dir`, `d` |
| file       | `file`, `fi`, `f`                |
| ssh remote | `remote`, `re`, `r`              |

# Config

| env                     | description                                    | required                                                                                     | default value                        |
| ----------------------- | ---------------------------------------------- | -------------------------------------------------------------------------------------------- | ------------------------------------ |
| `VSC_CODE_BIN`          | Command to open `VScode`                       | `yes`<br/> But the default value was packaged into installer, you can modify it by yourself. | `/usr/local/bin/code`                |
| `VSC_CODE_INSIDERS_BIN` | Command to open `VScode insiders`              | `yes`<br/> but the default value was packaged into installer, you can modify it by yourself. | `/usr/local/bin/code-insiders`       |
| `VSC_IDE_PATH`          | The data path of the VScode editor             | `no`                                                                                         | `~/Library/Application Support/Code` |
| `VSC_DIRECTORIES`       | Custom list of directories, separated by `,` . | `no`                                                                                         | `''`                                 |
