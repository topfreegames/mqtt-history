CREATE KEYSPACE IF NOT EXISTS chat
  WITH REPLICATION = {
    'class': 'SimpleStrategy',
    'replication_factor': 1
  };

CREATE TABLE IF NOT EXISTS chat.messages (
  id timeuuid, 
  topic varchar, 
  payload varchar, 
  bucket int, 
  PRIMARY KEY ((topic, bucket), id)
) WITH CLUSTERING ORDER BY (id DESC);
