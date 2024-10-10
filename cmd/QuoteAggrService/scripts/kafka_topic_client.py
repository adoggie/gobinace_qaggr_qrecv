import datetime

from kafka import KafkaConsumer

import quote_period_pb2

print("kafka cunsumer start..")
# 假设 topic 名为 'my_topic'
consumer = KafkaConsumer(
    'cc_quote1',
    group_id='groupA',  # 对于用户 A
    # bootstrap_servers='172.16.30.21:9092'
    bootstrap_servers='193.32.149.156:9092'
)
tradefile = open('tradefile.log', 'w')
print("kafka ready..")
for message in consumer:

    # print(datetime.datetime.now(), f"User A received message: {message.value}")
    pm = quote_period_pb2.PeriodMessage()
    print("<<", len(message.value))
    # continue
    # try:
    #     value = zlib.decompress(message.value)
    # except:
    #     continue
    ret = pm.ParseFromString(message.value)

    # print(f"message: {pm}")
    message = pm
    print(f"period:{message.period}, ts:{message.ts}, post_ts:{message.post_ts}, poster_id:{message.poster_id}")
    print(datetime.datetime.now(), " message period:", pm.period, " TS:", pm.ts)
    for symbol_info in message.symbol_infos:
        if symbol_info.symbol == 'ETHUSDT':
            print(
                f"  - symbol:{symbol_info.symbol} trades count:{len(symbol_info.trades)} incs count:{len(symbol_info.incs)}")
            for tr in symbol_info.trades:
                tradefile.write(
                    f"{datetime.datetime.now()} ETHUSDT trade cnt:{len(symbol_info.trades)} tradeid:{tr.timestamp} {tr.price} {tr.amount}\n")

                tradefile.flush()

        # for trade in symbol_info.trades:
        #     print(f"trade: {trade.timestamp}, {trade.side}, {trade.price}, {trade.amount}")
        # for inc in symbol_info.incs:
        #     print(f"inc: {inc.timestamp}, {inc.is_snapshot}, {inc.side}, {inc.price}, {inc.amount}")
        #
"""
KAFKA_CLUSTER_ID="$(bin/kafka-storage.sh random-uuid)"
bin/kafka-storage.sh format -t $KAFKA_CLUSTER_ID -c config/kraft/server.properties
bin/kafka-storage.sh format -t $KAFKA_CLUSTER_ID -c config/server.properties


https://kafka-python.readthedocs.io/en/master/
 
https://kafka.apache.org/documentation/#quickstart
pip install kafka-python
pip install protobuf
pip install upgrade protobuf 

bin/kafka-server-start.sh config/server.properties
bin/kafka-server-start.sh config/kraft/server.properties
bin/kafka-console-consumer.sh --topic cc_quote1 --from-beginning --bootstrap-server 172.16.30.21:9092
bin/kafka-console-consumer.sh --topic cc_quote1 --from-beginning --bootstrap-server 193.32.149.156:9092
bin/kafka-console-producer.sh --topic cc_quote1 --bootstrap-server 172.16.30.21:9092
bin/kafka-console-producer.sh --topic cc_quote1 --bootstrap-server 193.32.149.156:9092


bin/kafka-topics.sh --create --topic cc_quote1 --bootstrap-server 193.32.149.156:9092 --config max.message.bytes=20971520
 bin/kafka-topics.sh --delete --topic cc_quote --bootstrap-server 193.32.149.156:9092

一次读完所有group的topic消息
bin/kafka-console-consumer.sh --topic cc_quote1 --from-beginning --bootstrap-server 193.32.149.156:9092 --group groupA > /dev/null 
 
bin/kafka-consumer-groups.sh --bootstrap-server 193.32.149.156:9092 --describe --group groupA
bin/kafka-consumer-groups.sh --bootstrap-server 193.32.149.156:9092  --group groupA --topic cc_quote1 --describe
 bin/kafka-topics.sh --topic cc_quote1  --bootstrap-server 193.32.149.156:9092  --describe

 rebalance 
 https://www.cnblogs.com/yoke/p/11405397.html 
"""
