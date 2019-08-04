# gosudachi

gosudachiは日本語形態素解析器である[Sudachi](https://github.com/WorksApplications/Sudachi)のGo移植版です。

以下では、株式会社ワークスアプリケーションズ徳島人工知能NLP研究所が開発公開しているオリジナルのSudachiを「Java版Sudachi」「Java版」、Java版sudachi用の辞書ファイルを「Java版sudachi辞書」と表記します。

gosudachiは、Java版sudachiのバージョン0.3.0相当です。


## 特徴

現時点のJava版Sudachiが持つ機能や特徴をすべて移植しました。よって詳しい情報は[Java版の文書](https://github.com/WorksApplications/Sudachi)を参照してください。この文書にはGo版のみに該当する内容が記述されています。

-   Java版と同じコマンドラインオプション
-   Java版と同じく分割モード指定が可能
-   Java版と同じシステム提供プラグイン同梱
-   Java版と同等のプラグインの仕組みを提供
-   Java版と同じ設定ファイルが利用可能
-   ユーザー辞書の作成および利用が可能


## Java版とGo版の違い

-   辞書の文字列エンコード
-   設定ファイルに指定するプラグイン名
-   設定ファイルに辞書の文字列エンコードを指定する設定値を新設


### 辞書の文字列エンコードを変更した理由

Java版Sudachiは、辞書の作成時に文字列をUTF-16エンコードのバイト列として記録します。辞書を利用するときは、辞書ファイルをメモリにマップし、バイト列をそのまま（文字コード変換をせずに）文字列として扱います。

Goの文字列はUTF-8エンコードのバイト列であることが一般的です。GoでJavaと同様に辞書中のバイト列をそのまま文字列として扱うには、UTF-8エンコードで記録された辞書を準備する必要があります。

Go版ではシステム辞書作成ツールとして `dicbuilder` 、ユーザー辞書作成ツールとして `userdicbuilder` を準備しており、どちらもUTF-8エンコードの辞書を作成します。（UTF-16エンコードの辞書を作成することもできます。 `dicconv` を使って相互に変換することも可能です。）

ただし、UTF-8エンコードの辞書はUTF-16エンコードの辞書よりもサイズが大きくなります。以下の2点がその理由です。

-   日本語に使用される文字の多くが、1文字あたりUTF-16では2byte長であり、UTF-8では3byte長
-   文字列のバイト長を記録するための領域に2byteを使用する頻度が高い

UTF-8エンコードでのバイト長が127を超える文字列の場合、2byteを使用してバイト長を記録します。なお、UTF-16エンコードの辞書ではバイト長ではなくUTF-16表現でのint16配列の長さを記録しており、記録可能な文字列の長さはUTF-8の方が短くなります。

ちなみに辞書中に記録される文字列とは、品詞情報リストおよび単語情報です。

Go版においても、UTF-16エンコードの辞書を利用することが可能です。この場合、辞書から文字列を読み出す処理においてUTF-16からUTF-8への文字コード変換が行われます。利用する辞書のエンコードを設定ファイルに設定できます。


### 設定ファイルの違い

Go版でのみ利用できる設定値に関する記述です。


#### utf16String

`utf16String` が `true` になっている場合、UTF-16エンコードの辞書であると判断します。デフォルトはfalseです。

    {
        "systemDict" : "system_core_utf16.dic",
        "utf16String" : true,
        ...
    }


#### プラグイン名

Go版ではJava版の設定ファイルをそのまま利用することが可能ですが、プラグイン名に省略形を用いることもできます。

Java版と同様にデフォルトで利用できるプラグインは以下の7つがあります。省略形とはJavaのクラス階層を省いたプラグイン名です。また、設定先は `class` ではなく `name` にすることも可能です。

| 処理部分 | プラグイン   | プラグイン名                                              | 省略形                            |
|-------- |------------ |--------------------------------------------------------- |--------------------------------- |
| 入力テキスト修正 | 文字列正規化 | com.worksap.nlp.sudachi.DefaultInputTextPlugin            | DefaultInputTextPlugin            |
|          | 長音正規化   | com.worksap.nlp.sudachi.ProlongedSoundMarkInputTextPlugin | ProlongedSoundMarkInputTextPlugin |
| 未知語処理 | 1文字未知語  | com.worksap.nlp.sudachi.SimpleOovProviderPlugin           | SimpleOovProviderPlugin           |
|          | MeCab互換    | com.worksap.nlp.sudachi.MeCabOovProviderPlugin            | MeCabOovProviderPlugin            |
| 単語接続処理 | 品詞接続禁制 | com.worksap.nlp.sudachi.InhibitConnectionPlugin           | InhibitConnectionPlugin           |
| 出力解修正 | カタカナ未知語まとめ上げ | com.worksap.nlp.sudachi.JoinKatakanaOovPlugin             | JoinKatakanaOovPlugin             |
|          | 数詞まとめ上げ | com.worksap.nlp.sudachi.JoinNumericPlugin                 | JoinNumericPlugin                 |

    {
        "systemDict" : "system_core.dic",
        "inputTextPlugin" : [
            { "name" : "DefaultInputTextPlugin" },
            { "name" : "ProlongedSoundMarkInputTextPlugin",
              "prolongedSoundMarks": ["ー", "-", "⁓", "〜", "〰"],
              "replacementSymbol": "ー"}
        ],
        "oovProviderPlugin" : [
            { "name" : "MeCabOovProviderPlugin" },
            { "name" : "SimpleOovProviderPlugin",
              "oovPOS" : [ "補助記号", "一般", "*", "*", "*", "*" ],
              "leftId" : 5968,
              "rightId" : 5968,
              "cost" : 3857 }
        ],
        "pathRewritePlugin" : [
            { "name" : "JoinNumericPlugin",
              "joinKanjiNumeric" : true },
            { "name" : "JoinKatakanaOovPlugin",
              "oovPOS" : [ "名詞", "普通名詞", "一般", "*", "*", "*" ],
              "minLength" : 3
            }
        ]
    }


## Goへのポーティング指針

以下の指針のもと、移植作業を行っています。

1.  なるべくJavaのコードに似たような構成にする
    -   オリジナルに修正が入ったときに追随しやすいように

2.  Java版Sudachiと同じ設定ファイルが利用できるように

3.  Java版Sudachiのコマンドラインインターフェースも同じにする

4.  Java版Sudachi用に作成された辞書ファイルをGo版でも使えるように

5.  Java版Sudachi用の辞書が作れるように


## ビルド

プログラムと辞書を作成する方法です。


### プログラムのビルド

このリポジトリをcloneします。 cloneしたディレクトリに移動し、ビルドスクリプトを実行します。

    $ git clone https://github.com/msnoigrs/gosudachi
    $ cd gosudachi
    $ bash scripts/build.sh

distディレクトリにバイナリが作成されます。作成されるバイナリは以下の通りです。

-   **gosudachicli:** Sudachiコマンドライン
-   **dicbuilder:** システム辞書作成ツール
-   **userdicbuilder:** ユーザー辞書作成ツール
-   **printdic:** 辞書ファイルに登録されている単語リスト表示プログラム
-   **printdicheader:** 辞書ファイルヘッダ情報表示プログラム
-   **dicconv:** 辞書の文字列エンコードをUTF-16とUTF-8間で相互に変換するプログラム

ビルドスクリプトを使わない場合は、コマンドプロンプト上で以下を実行してください。Windowsでも作成可能です。

    $ git clone https://github.com/msnoigrs/gosudachi
    $ cd gosudachi/data
    $ go generate
    $ cd ..
    $ cd gosudachicli
    $ go build
    $ cd ..
    $ cd dicbuilder
    $ go build
    $ cd ..
    $ cd userdicbuilder
    $ go build
    $ cd ..
    $ cd printdic
    $ go build
    $ cd ..
    $ go printdicheader
    $ go build
    $ cd ..
    $ cd dicconv
    $ go build


### 辞書の作成

辞書のソースもJava版Sudachiのものを利用します。 [SudachiDict](https://github.com/WorksApplications/SudachiDict)をgithubからcloneした後、git lfs pullで取得します。 辞書のソースファイルは、 `small_lex.csv` と `core_lex.csv` と `notcore_lex.csv` の3つです。

辞書を作成するスクリプトを利用する場合、以下を実行してください。

    $ git clone https://github.com/WorksApplications/SudachiDict.git
    $ cd SudachiDict
    $ git lfs pull
    $ cd ../dist
    $ bash ../scripts/mksystemdic.sh ../SudachiDict

distディレクトリに `system_small.dic` 、 `system_core.dic` および `system_full.dic` ファイルが作成されます。

辞書作成スクリプトを使わない場合は、コマンドプロンプト上で以下を実行してください。

    $ dicbuilder -o system_small.dic -m matrix.def small_lex.csv
    $ dicbuilder -o system_core.dic -m matrix.def small_lex.csv core_lex.csv
    $ dicbuilder -o system_full.dic -m matrix.def small_lex.csv core_lex.csv notcore_lex.csv


## コマンド

Go版で提供するコマンドの説明です。


### gosudachicli

Sudachiコマンドラインです。オプションを指定せずに実行する場合、 `system_core.dic` ファイルが実行時のディレクトリに存在する必要があります。辞書ファイルの場所は設定ファイルに指定可能です。

    $ gosudachicli [-r conf] [-m mode] [-a] [-d] [-o output] [-j] [file...]


#### オプション

-   -r conf設定ファイルを指定
-   -s デフォルト設定を上書きする設定(json文字列)
-   -p リソースディレクトリ(設定ファイル内の各種リソースのベースディレクトリ、デフォルトは実行時ディレクトリ)
-   -m {A|B|C}分割モード
-   -a 読み、辞書形も出力
-   -d デバッグ情報の出力
-   -o 出力ファイル（指定がない場合は標準出力）
-   -f エラーを無視して処理を続行する
-   -j UTF-16エンコードの辞書ファイルを利用する


#### 出力例

    $ echo 東京都へ行く | gosudachicli
    東京都  名詞,固有名詞,地名,一般,*,*     東京都
    へ      助詞,格助詞,*,*,*,*     へ
    行く    動詞,非自立可能,*,*,五段-カ行,終止形-一般       行く
    EOS
    
    $ echo 東京都へ行く | gosudachicli -a
    東京都  名詞,固有名詞,地名,一般,*,*     東京都  東京都  トウキョウト
    へ      助詞,格助詞,*,*,*,*     へ      へ      エ
    行く    動詞,非自立可能,*,*,五段-カ行,終止形-一般       行く    行く    イク
    EOS
    
    $ echo 東京都へ行く | gosudachicli -m A
    東京    名詞,固有名詞,地名,一般,*,*     東京
    都      名詞,普通名詞,一般,*,*,*        都
    へ      助詞,格助詞,*,*,*,*     へ
    行く    動詞,非自立可能,*,*,五段-カ行,終止形-一般       行く
    EOS

-   **Java版:** com.worksap.nlp.sudachi.SudachiCommandLine


### dicbuilder

辞書ソースファイルからシステム辞書を作成します。デフォルトではUTF-8エンコードの辞書が作成されます。

    $ dicbuilder -o outputdic -m matrix.def [-d description] [-j] filecsv1 [filecsv2...]


#### オプション

-   -o 出力ファイル（必須）
-   -m matrix.defファイル（必須）
-   -d 辞書ヘッダ情報に埋め込む文字
-   -j UTF-16エンコードの辞書ファイルを生成する

-   **Java版:** com.worksap.nlp.sudachi.dictionary.DictionaryBuilder


### userdicbuilder

ユーザー辞書ソースファイルからユーザー辞書を作成します。デフォルトではUTF-8エンコードの辞書が作成されます。

    $ userdicbuilder -o outputdic -s systemdic [-d description] [-j] filecsv1 [filecsv2...]


#### オプション

-   -o 出力ファイル（必須）
-   -s システム辞書ファイル（必須）
-   -d 辞書ヘッダ情報に埋め込む文字
-   -j UTF-16エンコードの辞書ファイルを生成する

-   **Java版:** com.worksap.nlp.sudachi.dictionary.UserDictionaryBuilder


### printdic

辞書ファイルに登録されている単語リストを表示します。

    $ printdic [-s systemdic] [-j] inputdic


#### オプション

-   -s システム辞書ファイル（ユーザー辞書の情報を出力する場合に必要）
-   -j UTF-16エンコードの辞書を読み込み

-   **Java版:** com.worksap.nlp.sudachi.dictionary.DictionaryPrinter


### printdicheader

辞書ファイルのヘッダ情報を表示します。

    $ printdicheader inputdic

-   **java版:** com.worksap.nlp.sudachi.dictionary.DictionaryHeaderPrinter


### dicconv

辞書ファイルに記録されている文字列のエンコードを変換します。オプションを指定しない場合、UTF-16エンコード（Java版）からUTF-8エンコード（Go版）に変換します。

    $ dicconv [-o outputdic] [-j] inputdic


#### オプション

-   -o 出力ファイル、省略すると `out_utf16.dic` もしくは `out_utf8.dic` に出力
-   -j UTF-8エンコードからUTF-16エンコードに変換する


## ライセンス

Java版Sudachiと同じ[Apache License, Version2.0](http://www.apache.org/licenses/LICENSE-2.0.html)


## 謝辞

[Sudachi](https://github.com/WorksApplications/Sudachi)においてプログラムや辞書をOSSとして公開されている、株式会社ワークスアプリケーションズ徳島人工知能NLP研究所およびその開発者の方々に感謝いたします。
