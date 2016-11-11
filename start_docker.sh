#!/bin/sh

if [ "$MQTTHISTORY_API_TLS" == "true" ]
then
  echo -e $MQTTHISTORY_API_CERT > $MQTTHISTORY_API_CERTFILE
  echo -e $MQTTHISTORY_API_KEY > $MQTTHISTORY_API_KEYFILE
  export PORT=4443
else
  export PORT=5000
fi

/go/bin/mqtt-history start --bind 0.0.0.0 --port ${PORT} --config ${MQTTHISTORY_CONFIG_FILE}
