# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# NO CHECKED-IN PROTOBUF GENCODE
# source: quote_common.proto
# Protobuf Python Version: 5.27.3
"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import runtime_version as _runtime_version
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
_runtime_version.ValidateProtobufRuntimeVersion(
    _runtime_version.Domain.PUBLIC,
    5,
    27,
    3,
    '',
    'quote_common.proto'
)
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()


import quote_period_pb2 as quote__period__pb2


DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x12quote_common.proto\x12\x07message\x1a\x12quote_period.proto\"\xc4\x01\n\tDepthInfo\x12\x11\n\ttimestamp\x18\x01 \x01(\x03\x12\x10\n\x08trans_ts\x18\x02 \x01(\x03\x12\x17\n\x0f\x66irst_update_id\x18\x03 \x01(\x03\x12\x16\n\x0elast_update_id\x18\x04 \x01(\x03\x12\x1b\n\x13prev_last_update_id\x18\x05 \x01(\x03\x12!\n\x04\x62ids\x18\x06 \x03(\x0b\x32\x13.message.PriceLevel\x12!\n\x04\x61sks\x18\x07 \x03(\x0b\x32\x13.message.PriceLevel\"\xd4\x02\n\tKlineInfo\x12\x11\n\ttimestamp\x18\x01 \x01(\x03\x12\x12\n\nstart_time\x18\x02 \x01(\x03\x12\x10\n\x08\x65nd_time\x18\x03 \x01(\x03\x12\x0e\n\x06symbol\x18\x04 \x01(\t\x12\x10\n\x08interval\x18\x05 \x01(\t\x12\x16\n\x0e\x66irst_trade_id\x18\x06 \x01(\x03\x12\x15\n\rlast_trade_id\x18\x07 \x01(\x03\x12\x0c\n\x04open\x18\x08 \x01(\x01\x12\r\n\x05\x63lose\x18\t \x01(\x01\x12\x0c\n\x04high\x18\n \x01(\x01\x12\x0b\n\x03low\x18\x0b \x01(\x01\x12\x0e\n\x06volume\x18\x0c \x01(\x01\x12\x11\n\ttrade_num\x18\r \x01(\x03\x12\x10\n\x08is_final\x18\x0e \x01(\x08\x12\x14\n\x0cquote_volume\x18\x0f \x01(\x01\x12\x19\n\x11\x61\x63tive_buy_volume\x18\x10 \x01(\x01\x12\x1f\n\x17\x61\x63tive_buy_quote_volume\x18\x11 \x01(\x01\"\xc0\x01\n\x0c\x41ggTradeInfo\x12\x11\n\ttimestamp\x18\x01 \x01(\x03\x12\x0e\n\x06symbol\x18\x02 \x01(\t\x12\x1a\n\x12\x61ggregate_trade_id\x18\x03 \x01(\x03\x12\r\n\x05price\x18\x04 \x01(\x01\x12\x10\n\x08quantity\x18\x05 \x01(\x01\x12\x16\n\x0e\x66irst_trade_id\x18\x06 \x01(\x03\x12\x15\n\rlast_trade_id\x18\x07 \x01(\x03\x12\x12\n\ntrade_time\x18\x08 \x01(\x03\x12\r\n\x05maker\x18\t \x01(\x08\x42\x14Z\x12../message;messageb\x06proto3')

_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'quote_common_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
  _globals['DESCRIPTOR']._loaded_options = None
  _globals['DESCRIPTOR']._serialized_options = b'Z\022../message;message'
  _globals['_DEPTHINFO']._serialized_start=52
  _globals['_DEPTHINFO']._serialized_end=248
  _globals['_KLINEINFO']._serialized_start=251
  _globals['_KLINEINFO']._serialized_end=591
  _globals['_AGGTRADEINFO']._serialized_start=594
  _globals['_AGGTRADEINFO']._serialized_end=786
# @@protoc_insertion_point(module_scope)
