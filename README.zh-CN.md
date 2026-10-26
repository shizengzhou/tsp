# tsp (Terminal Settings Preview)

[English](README.md) | 简体中文

[![Release](https://github.com/shizengzhou/tsp/actions/workflows/release.yml/badge.svg)](https://github.com/shizengzhou/tsp/actions/workflows/release.yml)

终端配置的预览小工具集合：

- `tsp omz themes` — oh-my-zsh 主题浏览器，右侧实时预览提示符；回车通过 `omz theme set` 应用主题
- `tsp fonts` — 已安装字体家族浏览器，右侧显示样式与字形示例；回车把所选家族名输出到 stdout

## 环境要求

- Go 1.27.3+（从源码构建）
- `tsp omz themes`：`zsh` 与 oh-my-zsh（默认读取 `~/.oh-my-zsh`）
- `tsp fonts`：除二进制外无额外依赖

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

### 主题

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

### 字体

```bash
tsp fonts
```

左侧列出所有能解析的已安装字体家族：

- macOS：`~/Library/Fonts`、`/Library/Fonts`、`/System/Library/Fonts` 及其 `Supplemental` 目录
- Linux：`~/.fonts`、`~/.local/share/fonts`、`/usr/local/share/fonts`、`/usr/share/fonts`
- Windows：`%WINDIR%\Fonts`、`%LOCALAPPDATA%\Microsoft\Windows\Fonts`

右侧显示家族名、样式、带单行 ASCII 字形示例（与 kitty `choose-fonts` 相同）的 `preview` 区块，以及 PostScript 名和源文件。所选字体缺失的字符会被略去；ASCII 覆盖不足或没有可缩放轮廓的字体会退回文本示例。`Enter` 会把选中的家族名打印到 stdout，方便脚本读取：

```bash
font="$(tsp fonts)"
```

在能响应图形能力查询的终端中——Kitty graphics 协议（kitty、Ghostty、WezTerm 以及其他基于 libghostty 的终端）或 iTerm2 inline image 协议（iTerm2、开启 `terminal.integrated.enableImages` 的 VS Code）——示例会用所选字体栅格化成图片并贴到右侧，是真实预览。启动浏览器前会直接向终端查询能力；对不响应查询的终端（部分 VS Code 系终端只实现了 inline image 而不响应查询）再用环境变量兜底。图片背景透明，字形颜色会根据终端上报的背景色（无法获取时回退 `COLORFGBG`）自动选择深色或浅色，因此在明/暗主题下都清晰可读，也不会出现色块。其他终端与 tmux 会退回用终端当前字体绘制的文本示例，并在界面中说明。`TSP_GRAPHICS=0` 可强制文本回退，`TSP_GRAPHICS=1` 可强制启用图形（便于测试）。

| 按键 | 动作 |
| --- | --- |
| `↑` / `↓`、`j` / `k` | 移动选择 |
| `PgUp` / `PgDn` | 翻页 |
| `g` / `G`、`Home` / `End` | 跳到首尾 |
| 直接输入 或 `/` | 按家族名过滤 |
| `Esc` | 清空过滤（并退出输入状态） |
| `Enter` | 输出家族名并退出 |
| `q`、`Ctrl+C` | 退出 |

## 主题预览是怎么生成的

预览不是静态示例，而是用真实 zsh 渲染出来的：

1. 非交互启动 `zsh -f`，加载 oh-my-zsh 的 `lib/git.zsh`、`lib/prompt_info_functions.zsh`、`lib/theme-and-appearance.zsh`
2. 关闭 oh-my-zsh 的异步 git 提示，保证提示符里的分支/脏状态同步可得
3. `source` 目标主题文件，然后以 `print -P` 展开 `PROMPT` / `RPROMPT`
4. 在临时 `HOME` 下准备一个带未提交改动的 `~/dev/tsp` git 仓库，分别渲染「git 仓库 · dirty」与「非 git 目录」两种场景
5. 把预览栏宽度通过 `COLUMNS` 传给 zsh，使 `af-magic` 这类按列宽绘制分隔线的主题与实际宽度一致

渲染结果按「主题 + 宽度」缓存；选中主题时才渲染，最多并发 3 个 zsh 进程，单次超时 6 秒。

## 其它行为

- 两个浏览器都必须在交互式终端中运行；输出被管道接走时会报错
- `tsp omz themes` 在未安装 oh-my-zsh（或 `<zsh>/themes` 为空）时报错并以退出码 1 结束，可用 `--zsh` 指定实际路径
- 自定义目录不存在不影响使用，只是 Custom 页为空
- 主题预览是静态快照：依赖外部工具或运行时的内容（virtualenv、kube 上下文、时间等）可能显示为空或旧值，按 `r` 可重新渲染
- 设置主题只改写 `.zshrc`，不会影响当前已经打开的 shell
- `tsp fonts` 用 `q` 退出时不会输出任何内容；只有 `Enter` 会打印家族名
- 字体示例使用 `golang.org/x/image` 栅格化，不会应用连字、样式集等 OpenType 特性

## 发布

推送版本 tag 后，[release workflow](.github/workflows/release.yml) 会运行测试、构建 macOS/Linux/Windows（amd64/arm64）压缩包，并附带校验和发布：

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
internal/omz/preview.go 主题预览渲染器、临时 HOME、结果缓存
internal/omz/ui.go      主题浏览器界面
internal/omz/set.go     调用 omz theme set 应用主题
internal/omz/run.go     主题入口与终端检查
internal/fonts/font.go  已安装字体发现与家族分组
internal/fonts/ui.go    字体浏览器界面
internal/fonts/image.go 将字体示例栅格化为 PNG
internal/fonts/probe.go 向终端查询其支持的图形协议
internal/fonts/graphics.go 图形协议选择与图像转义序列
internal/fonts/run.go   字体入口与终端检查
```
