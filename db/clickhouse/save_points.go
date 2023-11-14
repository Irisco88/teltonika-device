package clickhouse

import (
	"context"
	"go.uber.org/zap"
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
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	batch, err := adb.ClickhouseConn.PrepareBatch(ctx, insertAvlPointQuery)
	if err != nil {
		//logger.Info("savePoints&&&&&&&&&&&&&&&&&&&&&&&&&&&&",
		//	zap.Any("2:", err),
		//)
		return err
	}
	for _, point := range points {
		gps := point.GetGps()
		//logger.Info("savePoints&&&&&&&&&&&&&&&&&&&&&&&&&&&&",
		//	zap.Any("3:", gps),
		//)
		elementMap := make(map[string]float64)
		//logger.Info("savePoints&&&&&&&&&&&&&&&&&&&&&&&&&&&&",
		//	zap.Any("40:", points),
		//)
		for _, element := range point.IoElements {

			elementMap[(element.ElementName)] = element.ElementValue
			//logger.Info("savePoints&&&&&&&&&&&&&&&&&&&&&&&&&&&&",
			//	zap.Any("4:", elementMap),
			//)
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
		//logger.Info("savePoints&&&&&&&&&&&&&&&&&&&&&&&&&&&&",
		//	zap.Any("5:", err),
		//)
		if err != nil {
			//logger.Info("savePoints&&&&&&&&&&&&&&&&&&&&&&&&&&&&",
			//	zap.Any("6:", err),
			//)
			return err
		}
	}
	//logger.Info("savePoints&&&&&&&&&&&&&&&&&&&&&&&&&&&&",
	//	zap.Any("7:", "*****"),
	//)
	return batch.Send()
}
