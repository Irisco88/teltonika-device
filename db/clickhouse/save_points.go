package clickhouse

import (
	"context"
	"time"

	pb "github.com/irisco88/protos/gen/device/v1"
)

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

		elementMap := make(map[string]float64)
		for _, element := range point.IoElements {
			elementMap[(element.ElementName)] = element.ElementValue
		}
		err := batch.Append(
			point.GetImei(),
			time.UnixMilli(int64(point.GetTimestamp())),
			point.Priority.String(),
			gps.GetLongitude(),
			gps.GetLatitude(),
			int16(gps.GetAltitude()),
			int16(gps.GetAngle()),
			uint8(gps.GetSatellites()),
			int16(gps.GetSpeed()),
			uint16(point.GetEventId()),
			elementMap,
		)
		if err != nil {
			return err
		}
	}
	return batch.Send()
}
