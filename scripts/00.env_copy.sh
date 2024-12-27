#!/bin/bash

example_path=${PWD}/.env.example
env_path=${PWD}/.env

# 判定
if [ ! -e $example_path ]; then
    echo ".env.exampleファイルが見つかりませんでした。クローンが失敗している可能性があります"
    exit 1
else
    # 判定(.env)が存在するか(存在する場合は終了コード１で終了)
    if [ -e $env_path ]; then
        echo ".envファイルが見つかりました、この操作は必要ないため終了します"
        exit 1
    # 存在しない場合はコピーを実行
    else
        echo ".envファイルが見つからなかったため、コピーを実行します"
        cp $example_path $env_path
        echo ".envファイルのコピーが完了しました。各種パラメーターは適宜変更してください"
        exit 0
    fi
fi