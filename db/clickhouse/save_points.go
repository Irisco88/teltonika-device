package clickhouse

import (
	"context"
	"github.com/packetify/teltonika-device/proto/pb"
	"time"
)

type AVLPointColumns struct {
	IMEI       string
	Timestamp  time.Time
	Priority   string
	Longitude  float64
	Latitude   float64
	Altitude   int16
	Angle      int16
	Satellites uint8
	Speed      int16
	IOElements map[uint16]int64
	EventID    uint16
}

const insertAvlPointQuery = `
	INSERT INTO 
	    avlpoints(imei, timestamp, priority, longitude, latitude, altitude, angle, satellites, speed,event_id, io_elements)
	VALUES (?,?,?,?,?,?,?,?,?,?,?);
`

// SaveAvlPoints saves avl points to clickhouse
func (adb *AVLDataBase) SaveAvlPoints(ctx context.Context, points []*pb.AVLData) error {
	batch, err := adb.ClickhouseConn.PrepareBatch(ctx, insertAvlPointQuery)
	if err != nil {
		return err
	}
	for _, point := range points {
		gps := point.GetGps()
		elementMap := make(map[uint16]int64)
		for _, element := range point.IoElements {
			elementMap[uint16(element.ElementId)] = element.Value
		}
		err := batch.AppendStruct(&AVLPointColumns{
			IMEI:       point.GetImei(),
			Timestamp:  time.UnixMilli(int64(point.GetTimestamp())),
			Priority:   point.Priority.String(),
			Longitude:  gps.GetLongitude(),
			Latitude:   gps.GetLatitude(),
			Altitude:   int16(gps.GetAltitude()),
			Satellites: uint8(gps.GetSatellites()),
			Speed:      int16(gps.GetSpeed()),
			EventID:    uint16(point.GetEventId()),
			IOElements: elementMap,
		})
		if err != nil {
			return err
		}
	}
	return batch.Send()
}
