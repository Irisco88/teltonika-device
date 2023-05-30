package server

import (
	"bytes"
	"fmt"
	"github.com/packetify/teltonika-device/proto/pb"
	"go.uber.org/zap"
	"net"
	"sync"
)

type Empty struct{}

type TeltonikaServer struct {
	listenAddr string
	ln         net.Listener
	quitChan   chan Empty
	wg         sync.WaitGroup
	log        *zap.Logger
}

const PRECISION = 10000000.0

type TcpServerInterface interface {
	Start()
	Stop()
}

var (
	_ TcpServerInterface = &TeltonikaServer{}
)

func NewServer(listenAddr string, logger *zap.Logger) TcpServerInterface {
	return &TeltonikaServer{
		listenAddr: listenAddr,
		quitChan:   make(chan Empty),
		wg:         sync.WaitGroup{},
		log:        logger,
	}
}

func (ts *TeltonikaServer) Start() {
	ln, err := net.Listen("tcp", ts.listenAddr)
	if err != nil {
		ts.log.Error("failed to listen", zap.Error(err))
		return
	}
	defer ln.Close()
	ts.ln = ln

	go ts.acceptConnections()
	ts.log.Info("server started",
		zap.String("ListenAddress", ts.listenAddr),
	)
	<-ts.quitChan
}

func (ts *TeltonikaServer) acceptConnections() {
	for {
		conn, err := ts.ln.Accept()
		if err != nil {
			ts.log.Error("accept connection error", zap.Error(err))
			continue
		}
		ts.log.Info("new Connection to the server", zap.String("Address", conn.RemoteAddr().String()))
		ts.wg.Add(1)
		go ts.HandleConnection(conn)
	}
}

func (ts *TeltonikaServer) HandleConnection(conn net.Conn) {
	defer conn.Close()
	defer ts.wg.Done()
	authenticated := false
	var imei string
	for {
		// Make a buffer to hold incoming data.
		buf := make([]byte, 2048)

		// Read the incoming connection into the buffer.
		size, err := conn.Read(buf)
		if err != nil {
			ts.log.Error("read failed", zap.Error(err))
			break
		}
		if !authenticated {
			imei, err = DecodeIMEI(buf[:size])
			if err != nil {
				ts.log.Error("decode imei failed", zap.Error(err))
				return
			}
			ts.log.Info("Data received",
				zap.String("ip", conn.RemoteAddr().String()),
				zap.Int("size", size),
				zap.String("imei", imei),
			)
			ts.ResponseAcceptMsg(conn)
			authenticated = true
			continue
		}
		points, err := parseData(buf, size, imei)
		if err != nil {
			ts.log.Error("Error while parsing data",
				zap.Error(err),
				zap.String("imei", imei),
			)
			break
		}
		go ts.LogPoints(points)
		conn.Write([]byte{0, 0, 0, uint8(len(points))})
	}
}

func (ts *TeltonikaServer) LogPoints(points []*pb.AVLData) {
	for _, p := range points {
		ts.log.Info("new packet",
			zap.String("IMEI", p.GetImei()),
			zap.Int64("timestamp", p.GetTimestamp()),
			zap.Any("gps", p.GetGps()),
			zap.Any("IOElements", p.GetIoElements()),
		)
	}
}

func (ts *TeltonikaServer) ResponseAcceptMsg(conn net.Conn) {
	conn.Write([]byte{1})
}

func (ts *TeltonikaServer) ResponseDecline(conn net.Conn) {
	conn.Write([]byte{0})
}

func parseData(data []byte, size int, imei string) ([]*pb.AVLData, error) {
	reader := bytes.NewBuffer(data)
	// fmt.Println("Reader Size:", reader.Len())

	// Header
	reader.Next(4)                                   // 4 Zero Bytes
	dataLength, err := streamToInt32(reader.Next(4)) // Header
	if err != nil {
		return nil, err
	}
	reader.Next(1)                                    // CodecID
	pointsNumber, err := streamToInt8(reader.Next(1)) // Number of Records
	fmt.Println("Length of data:", dataLength)

	points := make([]*pb.AVLData, pointsNumber)

	for i := int8(0); i < pointsNumber; i++ {
		timestamp, err := streamToTime(reader.Next(8)) // Timestamp
		priority, err := streamToInt8(reader.Next(1))  // Priority

		// GPS Element
		longitudeInt, err := streamToInt32(reader.Next(4)) // Longitude
		//longitude := float64(longitudeInt) / PRECISION
		latitudeInt, err := streamToInt32(reader.Next(4)) // Latitude
		//latitude := float64(latitudeInt) / PRECISION

		altitude, err := streamToInt16(reader.Next(2))  // Altitude
		angle, err := streamToInt16(reader.Next(2))     // Angle
		Satellites, err := streamToInt8(reader.Next(1)) // Satellites
		speed, err := streamToInt16(reader.Next(2))     // Speed

		if err != nil {
			fmt.Println("Error while reading GPS Element")
			break
		}

		points[i] = &pb.AVLData{
			Imei:      imei,
			Timestamp: timestamp,
			Priority:  pb.PacketPriority(priority),
			Gps: &pb.GPS{
				Longitude:  longitudeInt,
				Latitude:   latitudeInt,
				Altitude:   int32(altitude),
				Angle:      int32(angle),
				Speed:      int32(speed),
				Satellites: int32(Satellites),
			},
		}
		// IO Events Elements

		reader.Next(1) // ioEventID
		reader.Next(1) // total Elements

		for stage := 1; stage <= 4; stage++ {
			stageElements, err := streamToInt8(reader.Next(1))
			if err != nil {
				break
			}

			for elementIndex := int8(0); elementIndex < stageElements; elementIndex++ {
				elementID, err := streamToInt32(reader.Next(1)) // elementID
				if err != nil {
					return nil, err
				}
				var elementValue int64
				switch stage {
				case 1: // One byte IO Elements
					tmp, e := streamToInt8(reader.Next(1))
					if e != nil {
						return nil, e
					}
					elementValue = int64(tmp)
				case 2: // Two byte IO Elements
					tmp, e := streamToInt16(reader.Next(2))
					if e != nil {
						return nil, e
					}
					elementValue = int64(tmp)
				case 3: // Four byte IO Elements
					tmp, e := streamToInt32(reader.Next(4))
					if e != nil {
						return nil, e
					}
					elementValue = int64(tmp)
				case 4: // Eight byte IO Elements
					elementValue, err = streamToInt64(reader.Next(8))
				}
				points[i].IoElements = append(points[i].IoElements, &pb.IOElement{
					ElementId: elementID,
					Value:     elementValue,
				})
			}
		}

		if err != nil {
			fmt.Println("Error while reading IO Elements")
			break
		}
	}
	// Once finished with the records we read the Record Number and the CRC
	_, err = streamToInt8(reader.Next(1))  // Number of Records
	_, err = streamToInt32(reader.Next(4)) // CRC

	return points, nil
}

func (ts *TeltonikaServer) Stop() {
	ts.wg.Wait()
	ts.quitChan <- Empty{}
	ts.log.Info("stop server")
}
