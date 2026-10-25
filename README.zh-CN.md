# tsp (Terminal Settings Preview)

[English](README.md) | 简体中文

[![Release](https://github.com/shizengzhou/tsp/actions/workflows/release.yml/badge.svg)](https://github.com/shizengzhou/tsp/actions/workflows/release.yml)

终端配置的预览小工具集合。目前包含一个 oh-my-zsh 主题浏览器：

- 左侧列出全部主题，右侧实时渲染该主题的真实提示符（`PROMPT` + `RPROMPT`）
- 支持内置主题与自定义主题、按名称过滤、All / Custom 分类
- 回车调用 `omz theme set` 直接写回 `~/.zshrc`

## 环境要求

- Go 1.27.3+
- `zsh`（预览和设置主题都通过 zsh 执行）
- oh-my-zsh（默认从 `~/.oh-my-zsh` 读取）

## 安装

### 安装脚本

```bash
curl -fsSL https://raw.githubusercontent.com/shizengzhou/tsp/main/install.sh | bash
```

脚本会自动识别 macOS/Linux 与 amd64/arm64，校验 `checksums.txt`，并把 `tsp` 安装到 `~/.local/bin`。指定版本或安装位置：

```bash
curl -fsSL https://raw.githubusercontent.com/shizengzhou/tsp/main/install.sh \
  | bash -s -- --version v0.1.0 --dir /usr/local/bin
```

### Windows（PowerShell）

```powershell
irm https://raw.githubusercontent.com/shizengzhou/tsp/main/install.ps1 | iex
```

指定版本或安装位置：

```powershell
irm https://raw.githubusercontent.com/shizengzhou/tsp/main/install.ps1 -OutFile install.ps1
.\install.ps1 -Version v0.1.0 -InstallDir C:\Tools
```

脚本会下载 Windows zip 包、校验 `checksums.txt`，并把 `tsp.exe` 安装到 `%USERPROFILE%\.local\bin`。支持 Windows PowerShell 5.1+ 与 PowerShell 7。

> `tsp` 仍然需要 `PATH` 中有 `zsh` 与 oh-my-zsh（例如通过 MSYS2 安装）。使用 WSL 时，请直接在 WSL 内用 `install.sh` 安装 Linux 版本。

### 手动下载

每个 [release](https://github.com/shizengzhou/tsp/releases) 都会附带 macOS 与 Linux（tar.gz）和 Windows（zip）的预编译包，解压后把 `tsp`（或 `tsp.exe`）放到 `PATH` 即可。

### 从源码构建

```bash
go build -o tsp .
# 或安装到 $GOBIN
go install .
```

## 使用

```bash
tsp omz themes
```

指定安装路径与自定义主题目录：

```bash
tsp omz themes --zsh ~/.oh-my-zsh --custom ~/.config/zsh/custom
```

### 参数

| 参数 | 说明 | 默认值 |
| --- | --- | --- |
| `--zsh DIR` | oh-my-zsh 安装目录 | `$ZSH`，否则 `~/.oh-my-zsh` |
| `--custom DIR` | 自定义主题目录 | `$ZSH_CUSTOM`，否则 `<zsh>/custom` |

主题来源为 `<zsh>/themes/*.zsh-theme` 与自定义目录下的 `*.zsh-theme`（根目录及子目录）。同名主题以自定义目录为准。

### 快捷键

| 按键 | 动作 |
| --- | --- |
| `↑` / `↓`、`j` / `k` | 移动选择 |
| `PgUp` / `PgDn` | 翻页 |
| `g` / `G`、`Home` / `End` | 跳到首尾 |
| 直接输入 或 `/` | 按名称过滤（大小写不敏感） |
| `Esc` | 清空过滤（并退出输入状态） |
| `Tab`、`1` / `2`、`←` / `→` | 切换 All / Custom |
| `Enter` | 调用 `omz theme set` 应用当前主题 |
| `r` | 清除预览缓存并重新渲染 |
| `q`、`Ctrl+C` | 退出 |

`Enter` 会改写 `~/.zshrc` 中的 `ZSH_THEME` 行，并生成 `.zshrc.bck` 备份；需要重新打开 zsh（或 `source ~/.zshrc`）后生效。

## 预览是怎么生成的

预览不是静态示例，而是用真实 zsh 渲染出来的：

1. 非交互启动 `zsh -f`，加载 oh-my-zsh 的 `lib/git.zsh`、`lib/prompt_info_functions.zsh`、`lib/theme-and-appearance.zsh`
2. 关闭 oh-my-zsh 的异步 git 提示，保证提示符里的分支/脏状态同步可得
3. `source` 目标主题文件，然后以 `print -P` 展开 `PROMPT` / `RPROMPT`
4. 在临时 `HOME` 下准备一个带未提交改动的 `~/dev/tsp` git 仓库，分别渲染「git 仓库 · dirty」与「非 git 目录」两种场景
5. 把预览栏宽度通过 `COLUMNS` 传给 zsh，使 `af-magic` 这类按列宽绘制分隔线的主题与实际宽度一致

渲染结果按「主题 + 宽度」缓存；选中主题时才渲染，最多并发 3 个 zsh 进程，单次超时 6 秒。

## 其它行为

- 必须在交互式终端中运行；输出被管道接走时会报错 `tsp omz themes needs an interactive terminal`
- 未安装 oh-my-zsh（或 `<zsh>/themes` 为空）时报错并以退出码 1 结束，可用 `--zsh` 指定实际路径
- 自定义目录不存在不影响使用，只是 Custom 页为空
- 预览是静态快照：依赖外部工具或运行时的内容（virtualenv、kube 上下文、时间等）可能显示为空或旧值，按 `r` 可重新渲染
- 设置主题只改写 `.zshrc`，不会影响当前已经打开的 shell

## 发布

推送版本 tag 后，[release workflow](.github/workflows/release.yml) 会运行测试、构建 macOS/Linux（amd64/arm64）压缩包，并附带校验和发布：

```bash
git tag v0.1.0
git push origin v0.1.0
```

## 目录结构

```
main.go                 命令分发与帮助
install.sh              macOS/Linux 安装脚本
install.ps1             Windows 安装脚本
internal/omz/theme.go   主题发现（$ZSH/themes + $ZSH_CUSTOM）
internal/omz/preview.go 预览渲染器、临时 HOME、结果缓存
internal/omz/ui.go      bubbletea/lipgloss 界面
internal/omz/set.go     调用 omz theme set 应用主题
internal/omz/run.go     入口与终端检查
```
