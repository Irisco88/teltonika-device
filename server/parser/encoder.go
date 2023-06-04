package parser

type AVLData struct {
	Imei       string         `json:"imei,omitempty"`
	Timestamp  uint64         `json:"timestamp,omitempty"`
	Priority   PacketPriority `json:"priority,omitempty"`
	Gps        *GPS           `json:"gps,omitempty"`
	IoElements []*IOElement   `json:"io_elements,omitempty"`
	EventId    uint32         `json:"event_id,omitempty"`
}

type GPS struct {
	Longitude  float64 `json:"longitude,omitempty"`
	Latitude   float64 `json:"latitude,omitempty"`
	Altitude   int32   `json:"altitude,omitempty"`
	Angle      int32   `json:"angle,omitempty"`
	Satellites int32   `json:"satellites,omitempty"`
	Speed      int32   `json:"speed,omitempty"`
}

type IOElement struct {
	ElementId int32 ` json:"element_id,omitempty"`
	Value     int64 ` json:"value,omitempty"`
}

type PacketPriority uint8

const (
	priorityLow   PacketPriority = 0
	priorityHigh  PacketPriority = 1
	priorityPanic PacketPriority = 2
)
