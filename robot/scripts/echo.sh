#!/bin/bash
command=$1
shift

case "$command" in
  "echo")
    echo hello world
    curl 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=ca382666-138a-4f8a-9532-09055aa2b52f' -H 'Content-Type: application/json' -d '{"msgtype":"text","text":{"content":"hello world"}}'
    ;;
esac
