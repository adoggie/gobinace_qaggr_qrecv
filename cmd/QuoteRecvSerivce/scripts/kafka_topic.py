from kafka import KafkaConsumer

# 假设 topic 名为 'my_topic'
consumer = KafkaConsumer(
    'quickstart-events',
    group_id='groupA',  # 对于用户 A
    bootstrap_servers='172.16.30.21:9092'
)

for message in consumer:
    print(f"User A received message: {message.value.decode('utf-8')}")

"""

pip install kafka-python
pip install protobuf

bin/kafka-server-start.sh config/kraft/server.properties
bin/kafka-console-consumer.sh --topic quickstart-events --from-beginning --bootstrap-server 172.16.30.21:9092
bin/kafka-console-producer.sh --topic quickstart-events --bootstrap-server 172.16.30.21:9092

"""
