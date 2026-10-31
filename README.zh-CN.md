# tsp (Terminal Settings Preview)

[English](README.md) | 简体中文

[![Release](https://github.com/shizengzhou/tsp/actions/workflows/release.yml/badge.svg)](https://github.com/shizengzhou/tsp/actions/workflows/release.yml)

[![tsp 演示](tsp.jpg)](https://youtu.be/tB8dpq3Hw1c)

终端配置的预览小工具集合：

- `tsp omz themes` — oh-my-zsh 主题浏览器，右侧实时预览提示符；回车通过 `omz theme set` 应用主题
- `tsp omp themes` — oh-my-posh 主题浏览器，右侧实时预览提示符；回车改写 shell profile 中的 `oh-my-posh init` 行
- `tsp fonts` — 已安装等宽字体家族浏览器，右侧显示样式与字形示例；回车把所选家族名输出到 stdout，在 Alacritty、Ghostty 或 Windows Terminal 内则保存到终端配置
- `tsp themes` — Alacritty、WezTerm、Ghostty 与 Windows Terminal 配色主题浏览器，右侧预览调色板与代码；回车通过 `general.import`、`color_scheme`、`theme` 或 `colorScheme` 应用主题

## 环境要求

- Go 1.27.3+（从源码构建）
- `tsp omz themes`：`zsh` 与 oh-my-zsh（默认读取 `~/.oh-my-zsh`）
- `tsp omp themes`：`oh-my-posh` 在 `PATH` 中（主题来自 `$POSH_THEMES_PATH` 与安装器目录，含 Microsoft Store 包）
- `tsp fonts`：除二进制外无额外依赖

## 支持的终端

| 终端 | 字体预览 | 回车写入字体配置 | 配色主题浏览 |
| --- | --- | --- | --- |
| Alacritty | 文本示例（无图形协议） | 是，写入 `alacritty.toml` | 是 |
| Ghostty | 图片（Kitty graphics） | 是，写入其配置 | 是 |
| Windows Terminal | 文本示例（无图形协议） | 是，写入 `settings.json` | 是 |
| WezTerm | 图片（Kitty graphics） | 否，只打印家族名 | 是 |
| Kitty | 图片（Kitty graphics） | 否 | 否，直接报错退出 |
| iTerm2 | 图片（iTerm2 inline image） | 否 | 否，直接报错退出 |
| 开启 `terminal.integrated.enableImages` 的 VS Code | 图片（iTerm2 inline image） | 否 | 否 |
| Otty（Windows，基于 libghostty） | 图片（Kitty graphics，经 ConPTY 转发） | 否 | 否 |
| tmux | 文本示例 | 否 | 否 |
| 其它终端（Terminal.app、Konsole、GNOME Terminal 等） | 文本示例 | 否 | 否 |

- `tsp themes` 只在 Alacritty、WezTerm、Ghostty 与 Windows Terminal 内运行，因为其它终端看不到所选配色的效果；Windows Terminal 只列出 `settings.json` 的 `schemes` 中已有的配色。
- 在 tmux 等没有图形协议的终端里，字体示例用终端当前字体绘制，并显示提示 `cannot preview: kitty/iTerm2 image protocol not supported`。
- VS Code 的图片插件会把示例重采样到它自己的单元格网格，预览比原生终端更模糊。

## 安装

### 安装脚本

```bash
curl -fsSL https://raw.githubusercontent.com/shizengzhou/tsp/main/install.sh | bash
```

脚本会自动识别 macOS/Linux 与 amd64/arm64，用 `checksums.txt` 校验压缩包（缺失、未列出该压缩包或校验值不一致时拒绝安装），并把 `tsp` 安装到 `~/.local/bin`。指定版本或安装位置：

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

脚本会下载 Windows zip 包，用 `checksums.txt` 校验压缩包（缺失、未列出该压缩包或校验值不一致时拒绝安装），把 `tsp.exe` 安装到 `%USERPROFILE%\.local\bin`，并把这个目录追加到用户 `Path`（同时更新当前会话的 PATH），新开终端即可直接运行 `tsp`，无需注销。用 `-InstallDir` 自定义目录时，脚本会把该目录记到 `%LOCALAPPDATA%\tsp\install.txt`，卸载时无需再指定。支持 Windows PowerShell 5.1+ 与 PowerShell 7。

> `tsp` 仍然需要 `PATH` 中有 `zsh` 与 oh-my-zsh（例如通过 MSYS2 安装）。使用 WSL 时，请直接在 WSL 内用 `install.sh` 安装 Linux 版本。

### 手动下载

每个 [release](https://github.com/shizengzhou/tsp/releases) 都会附带 macOS 与 Linux（tar.gz）和 Windows（zip）的预编译包，解压后把 `tsp`（或 `tsp.exe`）放到 `PATH` 即可。

### 从源码构建

```bash
go build -o tsp .
# 或安装到 $GOBIN
go install .
```

Windows 上要保留 `.exe` 后缀，否则 PowerShell/CMD 无法执行该文件：

```powershell
go build -o tsp.exe .
# 或安装到 %GOBIN%
go install .
```

### 卸载

Windows：

```powershell
irm https://raw.githubusercontent.com/shizengzhou/tsp/main/uninstall.ps1 | iex
```

macOS/Linux：

```bash
curl -fsSL https://raw.githubusercontent.com/shizengzhou/tsp/main/uninstall.sh | bash
```

Windows 卸载脚本会自动读取安装时记录的目录（`%LOCALAPPDATA%\tsp\install.txt`），自定义安装位置也无需再传 `-InstallDir`；记录文件会随卸载一并清理。若目录已不在，可显式指定：`.\uninstall.ps1 -InstallDir C:\Tools`。

两个脚本都会删除二进制，并在安装目录变空后一并删掉该目录；Windows 上此时还会把它从用户 `Path`（以及当前会话）中移除，因此和 `tsp` 放在同一目录的其它工具不受影响。`tsp` 自己写入的文件——`ZSH_THEME` 行、`oh-my-posh` init 行、`alacritty.toml` 的 import、`wezterm.lua` 的 `color_scheme`、Ghostty 配置的 `theme` 与已下载的主题——不会被改动。

## 使用

### 终端配色主题（Alacritty、WezTerm、Ghostty 与 Windows Terminal）

```bash
tsp themes
```

在 Alacritty 内浏览 Alacritty 配置目录下的配色主题（`~/.config/alacritty/themes`，或 `$ALACRITTY_CONFIG_DIRECTORY/themes`）；首次运行且该目录不存在时，会先从 [iTerm2-Color-Schemes](https://github.com/mbadolato/iTerm2-Color-Schemes) 下载 Alacritty 版本的主题到该目录。在 WezTerm 内则列出 WezTerm 二进制自带的内置配色，无需联网。在 Ghostty 内按官方文档的搜索顺序列出 Ghostty 随包安装的主题和 `$XDG_CONFIG_HOME/ghostty/themes`（或 `~/.config/ghostty/themes`）下的自定义主题，同名时自定义主题优先。在 Windows Terminal 内列出 `settings.json` 的 `schemes` 中定义的配色；应用自带的配色只有在被写入该文件后才会出现，需要先在 Windows Terminal 设置里保存/添加。该命令必须在 Alacritty、WezTerm、Ghostty 或 Windows Terminal 内运行，因为在其他终端里看不到所选配色的效果，其余终端会直接报错退出。左侧是主题列表，右侧用主题自身的颜色预览调色板、前景/背景色和一段示例终端输出。

`Enter` 会应用该主题：在 Alacritty 内把 `alacritty.toml` 的 `general.import` 指向主题文件，并注释掉内联的 `[colors*]` 表（Alacritty 最后加载主配置文件，不注释的话内联颜色会覆盖导入的主题）；在 WezTerm 内写入 `wezterm.lua` 的 `color_scheme`；在 Ghostty 内写入 `theme = Name`，并注释掉会覆盖主题的 `background`、`foreground`、`palette`、`cursor-*` 与 `selection-*` 配置；在 Windows Terminal 内写入 `profiles.defaults.colorScheme`。四种情况都会保留 `.bck` 配置备份，按下 `Enter` 就会立即写入配置文件。Alacritty 的 YAML 配置不受支持，需先转换为 TOML；WezTerm 配置必须是 Lua；Ghostty 主题文件则与其配置一样使用 `key = value` 语法；Windows Terminal 使用带注释的 JSON（`settings.json`），编辑时保留原有注释与格式。

| 参数 | 说明 | 默认值 |
| --- | --- | --- |
| `--themes DIR` | Alacritty 主题目录 | `<Alacritty 配置目录>/themes` |
| `--config FILE` | 要修改的终端配置文件 | 配置目录下的 `alacritty.toml`，WezTerm 配置（`$WEZTERM_CONFIG_FILE`、`~/.config/wezterm/wezterm.lua`、`~/.wezterm.lua`），Ghostty 配置（`~/Library/Application Support/com.mitchellh.ghostty` 已有配置时用它，否则用 XDG 配置），或 Windows Terminal 的 `settings.json` |

按键：`↑`/`↓`、`j`/`k`、`PgUp`/`PgDn`、`g`/`G`，直接输入或 `/` 过滤，`Enter` 应用主题并退出，`Esc` 清空过滤，`q` 退出。

脚本场景可用 `tsp themes list` 逐行输出可用主题名，`tsp themes set NAME` 不打开浏览器直接应用，`tsp themes rand` 会用一行小转盘按各主题自身配色轮播候选并随机应用一个（仅在终端中播放动画）；三者都支持同样的 `--themes`/`--config` 参数，并作用于 tsp 当前运行的终端。

#### 每个工作日换一个主题

把下面内容保存为 `~/.local/bin/weekday-theme.sh` 并赋予执行权限（`chmod +x`）。它按 `date +%u`（1 = 周一）选择主题并通过 `tsp themes set` 应用。cron 没有终端环境，所以脚本显式导出 `TERM_PROGRAM` 告诉 tsp 该改哪个终端的配置；其它终端可改用 `WEZTERM_PANE=1` 或 `ALACRITTY_SOCKET=x`，主题名请以 `tsp themes list` 的输出为准。

```sh
#!/bin/sh
set -eu

# cron 没有终端环境，显式指定要修改哪个终端的配置。
export TERM_PROGRAM=ghostty

case "$(date +%u)" in
    1) theme="Catppuccin Latte" ;;
    2) theme="Catppuccin Frappe" ;;
    3) theme="Rose Pine Dawn" ;;
    4) theme="Everforest Dark Hard" ;;
    5) theme="Catppuccin Mocha" ;;
    *) exit 0 ;;
esac

exec "$HOME/.local/bin/tsp" themes set "$theme"
```

然后让它在每个工作日 08:30 执行：

```crontab
30 8 * * 1-5 "$HOME/.local/bin/weekday-theme.sh" >/dev/null 2>&1
```

### 版本与更新

```bash
tsp version
tsp update
```

`tsp update` 会检查 GitHub 最新 release，按当前系统/架构下载对应压缩包，用 release 的校验和验证后原地替换当前二进制。`checksums.txt` 缺失、未列出该压缩包或校验值不一致时一律拒绝更新，未经校验的下载不会被安装。若 tsp 所在目录不可写，可用 `sudo tsp update`，或重新运行安装脚本。`TSP_BASE_URL` 可覆盖仓库地址与下载地址，与安装脚本一致。

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

### oh-my-posh 主题

```bash
tsp omp themes
```

指定主题目录或其它 shell：

```bash
tsp omp themes --themes ~/.config/oh-my-posh/themes --shell zsh
```

### 参数

| 参数 | 说明 | 默认值 |
| --- | --- | --- |
| `--themes DIR` | 主题目录 | `$POSH_THEMES_PATH`、安装器目录与 Microsoft Store 包 |
| `--shell NAME` | 预览并写入配置的 shell | Windows 为 `pwsh`，其余为 `zsh` |

主题来自主题目录下的 `*.omp.json` 与 `*.json`。内嵌在 oh-my-posh 二进制里的 Store 版主题只按名字列出、也按名字引用，这样升级 oh-my-posh 后 profile 仍然有效。

预览由真实的 `oh-my-posh print` 渲染：使用临时 home 与一个带未提交改动的 git 仓库，因此会同时展示「仓库」和「普通目录」两种场景。预览按主题名与宽度缓存，选中时渲染，最多 3 个 oh-my-posh 进程并发，单次渲染超时 5 秒。

### 快捷键

| 按键 | 动作 |
| --- | --- |
| `↑` / `↓`、`j` / `k` | 移动选择 |
| `PgUp` / `PgDn` | 翻页 |
| `g` / `G`、`Home` / `End` | 跳到首尾 |
| 直接输入 或 `/` | 按名称过滤（大小写不敏感） |
| `Esc` | 清空过滤（并退出输入状态） |
| `Enter` | 把当前主题写入 shell profile |
| `r` | 清除预览缓存并重新渲染 |
| `q`、`Ctrl+C` | 退出 |

`Enter` 会改写 profile 中的 `oh-my-posh init` 行（插入或替换 `--config <主题>`），并生成 `<profile>.bck` 备份；需要重新打开 shell 后生效。

### 字体

```bash
tsp fonts
```

左侧列出所有能解析的已安装等宽字体家族，比例字体会被跳过：

- macOS：`~/Library/Fonts`、`/Library/Fonts`、`/System/Library/Fonts` 及其 `Supplemental` 目录
- Linux：`~/.fonts`、`~/.local/share/fonts`、`/usr/local/share/fonts`、`/usr/share/fonts`
- Windows：`%WINDIR%\Fonts`、`%LOCALAPPDATA%\Microsoft\Windows\Fonts`

右侧显示家族名、样式、带 ASCII 字形示例的 `preview` 区块，以及 PostScript 名和源文件。所选字体缺失的字符会被略去；ASCII 覆盖不足或没有可缩放轮廓的字体会退回文本示例。`Enter` 会把选中的家族名打印到 stdout；在 Alacritty 内则直接把所选家族保存到 Alacritty 配置（`alacritty.toml`）：自动匹配 normal、bold、italic、bold italic 四个样式，以统一的内联写法写入 `[font]` 表（例如 `normal = { family = 'Maple Mono NF', style = 'Regular' }`），并生成 `<config>.bck` 备份；Alacritty 会自动重载配置。旧版本写出的 `[font.normal]` 等独立表会一并合并成这种写法。家族缺少某个样式时不会写入对应项，并会清掉配置里遗留的同名字体（例如把 normal 换成 Cascadia Mono NF 后，其它样式里旧的 JetBrainsMono Nerd Font 会被移除），由 Alacritty 自行合成。在 Ghostty 内，`Enter` 会写入 `font-family`、`font-family-bold`、`font-family-italic` 与 `font-family-bold-italic`；每个列表都会先重置，因为 Ghostty 会把后面设置的值追加到前面配置文件的列表上；家族缺少的样式保持为空，由 Ghostty 从常规字体派生。在 Windows Terminal 内，`Enter` 会把 `settings.json` 的 `profiles.defaults.font.face` 改为所选家族（Windows Terminal 只有单一字面设置，粗体/斜体由它自行派生），并保留 `settings.json.bck` 备份；Windows Terminal 会自动重载设置。其它终端仍只打印家族名；配置或目录不存在时会新建。

在能响应图形能力查询的终端中——Kitty graphics 协议（Kitty、Ghostty、WezTerm 以及其他基于 libghostty 的终端）或 iTerm2 inline image 协议（iTerm2、开启 `terminal.integrated.enableImages` 的 VS Code）——示例会用所选字体栅格化成图片并贴到右侧，是真实预览。启动浏览器前会直接向终端查询能力；对不响应查询的终端（部分 VS Code 系终端只实现了 inline image 而不响应查询）再用环境变量兜底。图片背景透明，字形颜色会根据终端上报的背景色（无法获取时回退 `COLORFGBG`）自动选择深色或浅色，因此在明/暗主题下都清晰可读，也不会出现色块。终端上报单元格像素尺寸时（`TIOCGWINSZ` 或 `CSI 16 t`），示例会按该尺寸栅格化，字形大小与终端自身文字一致，而不是被终端缩放。其他终端与 tmux 无法预览图片，会退回用终端当前字体绘制的文本示例，并显示英文提示 `cannot preview: kitty/iTerm2 image protocol not supported`。Windows 上，控制台程序的输出要经过 ConPTY，而 ConPTY 通常会丢弃图形转义序列，因此只有自带 ConPTY 并会转发这些序列的终端才能显示图片预览。

| 按键 | 动作 |
| --- | --- |
| `↑` / `↓`、`j` / `k` | 移动选择 |
| `PgUp` / `PgDn` | 翻页 |
| `g` / `G`、`Home` / `End` | 跳到首尾 |
| 直接输入 或 `/` | 按家族名过滤 |
| `Esc` | 清空过滤（并退出输入状态） |
| `Enter` | 输出家族名并退出（在 Alacritty / Ghostty / Windows Terminal 内则写入配置） |
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

- 所有浏览器都必须在交互式终端中运行；输出被管道接走时会报错
- `tsp themes` 必须在 Alacritty、WezTerm、Ghostty 或 Windows Terminal 内运行（Alacritty 通过 `ALACRITTY_SOCKET`/`ALACRITTY_WINDOW_ID`、`TERM_PROGRAM=Alacritty` 或 `TERM` 以 `alacritty` 开头识别，Windows 上 ConPTY 会改写 `TERM`，因此改为检查父进程链中是否有 `alacritty.exe`；WezTerm 通过 `TERM_PROGRAM=WezTerm`、`WEZTERM_PANE` 或 `WEZTERM_EXECUTABLE` 识别；Ghostty 通过 `TERM_PROGRAM=ghostty`、`TERM` 以 `xterm-ghostty` 开头，或 `GHOSTTY_RESOURCES_DIR`/`GHOSTTY_BIN_DIR` 识别；Windows Terminal 在 Windows 构建下通过 `WT_SESSION` 识别），否则直接报错退出
- `tsp themes` 只解析 Alacritty 的 TOML 配色文件；`.yml` 配置会被拒绝并提示先迁移到 TOML；WezTerm 配置必须是 Lua；Ghostty 主题按官方文档从两个目录查找，使用普通 `key = value` 语法；Windows Terminal 的配色来自 `settings.json` 的 `schemes`（带注释的 JSON）
- Alacritty 首次运行 `tsp themes` 需要联网获取并下载主题，下载失败时不会留下 themes 目录；WezTerm 使用二进制自带的内置配色，Ghostty 的主题随应用安装，Windows Terminal 读取本地设置，均无需联网
- `tsp omz themes` 在未安装 oh-my-zsh（或 `<zsh>/themes` 为空）时报错并以退出码 1 结束，可用 `--zsh` 指定实际路径
- `tsp omp themes` 在 `oh-my-posh` 不在 `PATH`、或找不到任何主题时报错并以退出码 1 结束，可用 `--themes` 指定实际目录
- 自定义目录不存在不影响使用，只是 Custom 页为空
- 主题预览是静态快照：依赖外部工具或运行时的内容（virtualenv、kube 上下文、时间等）可能显示为空或旧值，按 `r` 可重新渲染
- 设置主题只改写 `.zshrc` 或 shell profile，不会影响当前已经打开的 shell
- `tsp fonts` 用 `q` 退出时不会输出任何内容；只有 `Enter` 会打印家族名（在 Alacritty / Ghostty / Windows Terminal 内则是写入终端配置并输出保存路径）
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
uninstall.sh            macOS/Linux 卸载脚本
uninstall.ps1           Windows 卸载脚本
scripts/verify-install.sh 用本地假 release 验证 install.sh
scripts/verify-install.ps1 PowerShell 下验证 install.ps1
internal/update/update.go 从 GitHub release 自更新
internal/fsutil/write.go 原子写入并保留原权限的配置写入辅助
internal/omz/theme.go   主题发现（$ZSH/themes + $ZSH_CUSTOM）
internal/omz/preview.go 主题预览渲染器、临时 HOME、结果缓存
internal/omz/ui.go      主题浏览器界面
internal/omz/set.go     调用 omz theme set 应用主题
internal/omz/run.go     主题入口与终端检查
internal/themebrowser/theme.go 两个浏览器共用的主题数据结构与颜色工具
internal/themebrowser/ui.go 共用的配色主题浏览器界面
internal/themebrowser/run.go 浏览器主循环与交互式终端检查
internal/alacritty/theme.go 主题发现与 TOML 颜色解析
internal/alacritty/download.go 首次运行从 iTerm2-Color-Schemes 下载
internal/alacritty/set.go 写入 general.import 并注释内联颜色
internal/alacritty/run.go alacritty 主题入口、首次运行准备
internal/wezterm/schemes.go 从 wezterm 二进制读取内置配色
internal/wezterm/config.go 写入 wezterm.lua 的 color_scheme
internal/wezterm/detect.go wezterm 环境识别
internal/wezterm/run.go wezterm 主题入口
internal/ghostty/themes.go ghostty 主题发现与 key = value 解析
internal/ghostty/config.go 写入 ghostty 配置的 theme
internal/ghostty/font.go 写入 ghostty 配置的字体家族
internal/ghostty/detect.go ghostty 环境识别
internal/ghostty/run.go ghostty 主题入口
internal/windowsterminal/detect.go Windows Terminal 识别与 settings.json 定位
internal/windowsterminal/font.go 写入 profiles.defaults.font.face
internal/windowsterminal/scheme.go 读取 schemes 并写入 profiles.defaults.colorScheme
internal/omp/theme.go   oh-my-posh 主题发现（含 Store 包目录）
internal/omp/preview.go oh-my-posh 主题预览渲染器、临时 HOME、结果缓存
internal/omp/ui.go      oh-my-posh 主题浏览器界面
internal/omp/set.go     改写 shell profile 中的 init 行
internal/omp/run.go     oh-my-posh 主题入口与终端检查
internal/fonts/font.go  已安装字体发现与家族分组
internal/fonts/ui.go    字体浏览器界面
internal/fonts/image.go 将字体示例栅格化为 PNG
internal/fonts/probe.go 向终端查询其支持的图形协议
internal/fonts/graphics.go 图形协议选择与图像转义序列
internal/fonts/run.go   字体入口与终端检查
```
