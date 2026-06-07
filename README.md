# cpt

AtCoder 向け C++ CLI ツール。テンプレート展開・コンパイル実行・テスト・スニペット管理をまとめて行います。

## 前提条件

以下のツールがインストール済みであることを前提とします。

- [oj](https://github.com/online-judge-tools/oj) — テストケースのダウンロード・実行
- [acc](https://github.com/Tatamo/atcoder-cli) — AtCoder コンテストの管理

## インストール

```bash
go install github.com/zuka-yuma/cpt@latest
```

`$HOME/go/bin` に `cpt` が入ります。PATH が通っていない場合は `~/.zprofile` に追加してください。

```bash
export PATH="$PATH:$HOME/go/bin"
```

### ソースからビルド

```bash
git clone https://github.com/zuka-yuma/cpt.git
cd cpt
go build -o cpt .
mv cpt /usr/local/bin/
```

## コマンド

### `cpt new <filename>`

テンプレートから `.cpp` ファイルを作成します。

```bash
cpt new main      # main.cpp を作成
```

### `cpt run <filename> [args]`

コンパイルして実行します。バイナリは残りません。

```bash
cpt run main.cpp
cpt run main.cpp < input.txt
```

### `cpt ac test`

コンパイルして `oj test` でテストを実行します。テストケースは `tests/` ディレクトリを参照します。

```bash
cpt ac test
cpt ac test -src a.cpp -d tests
```

### `cpt snippet`

スニペットを `~/.config/cpt/snippets/` で管理します。

```bash
cpt snippet list                      # 一覧表示
cpt snippet add <name>                # 追加（エディタが開く）
cpt snippet add -scope local <name>   # local スコープで追加
cpt snippet show <name>               # 内容確認
cpt snippet edit <name>               # 編集
cpt snippet insert <name>             # main.cpp に挿入
cpt snippet insert <name> <file>      # 対象ファイルを指定して挿入
cpt snippet delete <name>             # 削除（確認プロンプトあり）
cpt snippet delete -y <name>          # 確認なしで削除
```

**スニペットのスコープ:**
- `global` — `// @snippet:global` の位置に挿入（`main` の外）
- `local` — `// @snippet:local` の位置に挿入（`main` の中）

同じスニペットを2回挿入しようとするとスキップされます。

## 環境変数

| 変数 | デフォルト | 用途 |
|---|---|---|
| `CPT_CXX` | `g++-15` | コンパイラ |
| `CPT_CXXFLAGS` | `-std=gnu++23 -O2 -Wall` | コンパイルフラグ |
| `EDITOR` | `vi` | スニペット編集エディタ |

VS Code を使う場合: `export EDITOR="code --wait"`
