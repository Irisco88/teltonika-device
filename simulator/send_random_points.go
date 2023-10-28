package simulator

import (
	"github.com/irisco88/teltonika-device/parser"
	"time"
)

func (td *TrackerDevice) SendRandomPoints() {
	defer td.Stop()
	if err := td.AuthenticateIMEI(td.conn, td.imei); err != nil {
		td.log.Println("error for IMEI authentication:", err)
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
			td.log.Println("failed to send points:", err)
			return
		}

		td.log.Printf("Sent %d points to the server\n", numberOfPoints)
		for _, p := range points {
			td.log.Printf("%+v\n", p)
		}
		td.log.Println("######################")

		time.Sleep(time.Second * 3)
	}
}
