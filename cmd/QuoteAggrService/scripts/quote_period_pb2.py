# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# NO CHECKED-IN PROTOBUF GENCODE
# source: quote_period.proto
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
    'quote_period.proto'
)
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()




DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x12quote_period.proto\x12\x07message\"\xd1\x01\n\x16IncrementOrderBookInfo\x12\x11\n\ttimestamp\x18\x01 \x01(\x03\x12\x10\n\x08trans_ts\x18\x02 \x01(\x03\x12\x17\n\x0f\x66irst_update_id\x18\x03 \x01(\x03\x12\x16\n\x0elast_update_id\x18\x04 \x01(\x03\x12\x1b\n\x13prev_last_update_id\x18\x05 \x01(\x03\x12!\n\x04\x62ids\x18\x06 \x03(\x0b\x32\x13.message.PriceLevel\x12!\n\x04\x61sks\x18\x07 \x03(\x0b\x32\x13.message.PriceLevel\"\xbd\x01\n\tTradeInfo\x12\x11\n\ttimestamp\x18\x01 \x01(\x03\x12\x10\n\x08trade_id\x18\x02 \x01(\x03\x12\r\n\x05price\x18\x03 \x01(\x01\x12\x0b\n\x03qty\x18\x04 \x01(\x01\x12\x16\n\x0e\x62uyer_order_id\x18\x05 \x01(\x03\x12\x17\n\x0fseller_order_id\x18\x06 \x01(\x03\x12\x10\n\x08trade_ts\x18\x07 \x01(\x03\x12\x16\n\x0eis_buyer_maker\x18\x08 \x01(\x08\x12\x14\n\x0cplace_holder\x18\t \x01(\x08\"+\n\nPriceLevel\x12\r\n\x05price\x18\x01 \x01(\x01\x12\x0e\n\x06\x61mount\x18\x02 \x01(\x01\"\x91\x01\n\x0cSnapShotInfo\x12\x16\n\x0elast_update_id\x18\x01 \x01(\x03\x12\x11\n\ttimestamp\x18\x02 \x01(\x03\x12\x10\n\x08trade_ts\x18\x03 \x01(\x03\x12!\n\x04\x62ids\x18\x04 \x03(\x0b\x32\x13.message.PriceLevel\x12!\n\x04\x61sks\x18\x05 \x03(\x0b\x32\x13.message.PriceLevel\"\x95\x01\n\nSymbolInfo\x12\x0e\n\x06symbol\x18\x01 \x01(\t\x12\"\n\x06trades\x18\x02 \x03(\x0b\x32\x12.message.TradeInfo\x12-\n\x04incs\x18\x03 \x03(\x0b\x32\x1f.message.IncrementOrderBookInfo\x12$\n\x05snaps\x18\x04 \x03(\x0b\x32\x15.message.SnapShotInfo\"z\n\rPeriodMessage\x12\x0e\n\x06period\x18\x01 \x01(\x03\x12\n\n\x02ts\x18\x02 \x01(\x03\x12\x0f\n\x07post_ts\x18\x03 \x01(\x03\x12\x11\n\tposter_id\x18\x04 \x01(\t\x12)\n\x0csymbol_infos\x18\x05 \x03(\x0b\x32\x13.message.SymbolInfoB\x14Z\x12../message;messageb\x06proto3')

_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'quote_period_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
  _globals['DESCRIPTOR']._loaded_options = None
  _globals['DESCRIPTOR']._serialized_options = b'Z\022../message;message'
  _globals['_INCREMENTORDERBOOKINFO']._serialized_start=32
  _globals['_INCREMENTORDERBOOKINFO']._serialized_end=241
  _globals['_TRADEINFO']._serialized_start=244
  _globals['_TRADEINFO']._serialized_end=433
  _globals['_PRICELEVEL']._serialized_start=435
  _globals['_PRICELEVEL']._serialized_end=478
  _globals['_SNAPSHOTINFO']._serialized_start=481
  _globals['_SNAPSHOTINFO']._serialized_end=626
  _globals['_SYMBOLINFO']._serialized_start=629
  _globals['_SYMBOLINFO']._serialized_end=778
  _globals['_PERIODMESSAGE']._serialized_start=780
  _globals['_PERIODMESSAGE']._serialized_end=902
# @@protoc_insertion_point(module_scope)
