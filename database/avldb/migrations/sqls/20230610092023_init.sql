-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE TABLE avlpoints
(
    imei        String,
    timestamp   DateTime64(3),
    priority    Enum8('PACKET_PRIORITY_LOW' = 0, 'PACKET_PRIORITY_HIGH' = 1, 'PACKET_PRIORITY_PANIC' = 2),
    longitude   Float64,
    latitude    Float64,
    altitude    Int16,
    angle       Int16,
    satellites  UInt8,
    speed       Int16,
    io_elements Map(UInt16, Int64),
    event_id    UInt16
) ENGINE = MergeTree()
      ORDER BY (timestamp);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS avlpoints
-- +goose StatementEnd
