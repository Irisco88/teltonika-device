# teltonika device service
teltonika device service includes a `parser` to parse teltonika packets in `codec8 extended` protocol on `tcp` connection. this device service decodes `avl` data packets and saves all received points to a time serries database which in this case is `clickhouse`. it also publishes decoded points on nats subjects for each device using their `imei` 

## Avl data structure
this is avl point structure in `protobuf` format
```protobuf
message AVLData {
  string imei = 1;
  uint64 timestamp = 2; //millisecond
  PacketPriority priority = 3;
  GPS gps = 4;
  repeated IOElement io_elements = 5;
  uint32 event_id = 6;
}

message GPS {
  double longitude = 1;
  double latitude = 2;
  int32 altitude = 3;
  int32 angle = 4;
  int32 satellites = 5;
  int32 speed = 6;
}

message IOElement {
  int32 element_id = 1;
  int64 value = 2;
}

enum PacketPriority {
  PACKET_PRIORITY_LOW = 0;
  PACKET_PRIORITY_HIGH = 1;
  PACKET_PRIORITY_PANIC = 2;
}
```
## Build
```sh
just build
just upx #build and compress binary
```
## Deploy
deploy device service and its dependencies using provided docker compose file.
```sh
just dcompose-up
```
## TODO
+ save rawdata
- add grpc api to get `points` history
+ add grpc api to get `last-points`
- add `gotest` tools to justfile
+ fix server test for mock
- add support to `codec8` standard and others
+ check crc