package simulator

import (
	"github.com/openfms/teltonika-device/parser"
	"time"
)

func (td *TrackerDevice) SendRandomPoints() {
	defer td.Stop()
	if err := td.AuthenticateIMEI(td.conn, td.imei); err != nil {
		return
	}
	for {
		numberOfPoints := getRandomInt(1, 3)
		points := make([]*parser.AVLData, 0)
		for i := 0; i < numberOfPoints; i++ {
			point := generateRandomAVLData()
			points = append(points, point)
		}
		if err := td.SendPoints(td.conn, points); err != nil {
			td.log.Println("failed to send points", err)
			return
		}
		time.Sleep(time.Second * 3)
	}
}