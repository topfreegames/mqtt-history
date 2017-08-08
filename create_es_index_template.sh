#!/bin/sh

echo 'Create chat index template: '
curl -XPUT 'localhost:9123/_template/chat' -d '{"order": 0,"template": "chat-*","settings": {"index": {"number_of_replicas": "3"}},"mappings": {"message": {"properties": {"payload": {"type": "keyword"},"topic": {"type": "keyword"},"timestamp": {"format": "strict_date_optional_time||epoch_millis","type": "date"}}}},"aliases": {}}'

echo ''
echo 'Delete chat index: '
curl -XDELETE 'http://localhost:9123/chat-*'

# echo ''
# echo 'Create chat index (now with the correct index): '
# curl -XPOST 'http://localhost:9123/chat'

echo ''
