package server

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"go.uber.org/zap"
	"net"
	"sync"
	"time"
)

type Empty struct{}

type TeltonikaServer struct {
	listenAddr string
	ln         net.Listener
	quitChan   chan Empty
	wg         sync.WaitGroup
	log        *zap.Logger
}

// Location Struct for Mongo GeoJSON
type Location struct {
	Type        string
	Coordinates []float64
}

// Record Schema
type Record struct {
	Imei     string
	Location Location
	Time     time.Time
	Angle    int16
	Speed    int16
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

func (s *TeltonikaServer) Start() {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		s.log.Error("failed to listen", zap.Error(err))
		return
	}
	defer ln.Close()
	s.ln = ln

	go s.acceptConnections()
	s.log.Info("server started",
		zap.String("ListenAddress", s.listenAddr),
	)
	<-s.quitChan
}

func (s *TeltonikaServer) acceptConnections() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			s.log.Error("accept connection error", zap.Error(err))
			continue
		}
		s.log.Info("new Connection to the server", zap.String("Address", conn.RemoteAddr().String()))
		s.wg.Add(1)
		go s.HandleConnection(conn)
	}
}

func (s *TeltonikaServer) HandleConnection(conn net.Conn) {
	defer conn.Close()
	defer s.wg.Done()
	var b []byte
	var imei string
	knownIMEI := true
	step := 1

	for {
		// Make a buffer to hold incoming data.
		buf := make([]byte, 2048)

		// Read the incoming connection into the buffer.
		size, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading:", err.Error())
			break
		}

		// Send a response if known IMEI and matches IMEI size
		if knownIMEI {
			b = []byte{1} // 0x01 if we accept the message

			message := hex.EncodeToString(buf[:size])
			fmt.Println("----------------------------------------")
			fmt.Println("Data From:", conn.RemoteAddr().String())
			fmt.Println("Size of message: ", size)
			fmt.Println("Message:", message)
			fmt.Println("Step:", step)

			switch step {
			case 1:
				step = 2
				imei = message
				conn.Write(b)
			case 2:
				elements, err := parseData(buf, size, imei)
				if err != nil {
					fmt.Println("Error while parsing data", err)
					break
				}

				conn.Write([]byte{0, 0, 0, uint8(len(elements))})
			}

		} else {
			b = []byte{0} // 0x00 if we decline the message

			conn.Write(b)
			break
		}
	}
}

func parseData(data []byte, size int, imei string) (elements []Record, err error) {
	reader := bytes.NewBuffer(data)
	// fmt.Println("Reader Size:", reader.Len())

	// Header
	reader.Next(4)                                    // 4 Zero Bytes
	dataLength, err := streamToInt32(reader.Next(4))  // Header
	reader.Next(1)                                    // CodecID
	recordNumber, err := streamToInt8(reader.Next(1)) // Number of Records
	fmt.Println("Length of data:", dataLength)

	elements = make([]Record, recordNumber)

	var i int8 = 0
	for i < recordNumber {
		timestamp, err := streamToTime(reader.Next(8)) // Timestamp
		reader.Next(1)                                 // Priority

		// GPS Element
		longitudeInt, err := streamToInt32(reader.Next(4)) // Longitude
		longitude := float64(longitudeInt) / PRECISION
		latitudeInt, err := streamToInt32(reader.Next(4)) // Latitude
		latitude := float64(latitudeInt) / PRECISION

		reader.Next(2)                              // Altitude
		angle, err := streamToInt16(reader.Next(2)) // Angle
		reader.Next(1)                              // Satellites
		speed, err := streamToInt16(reader.Next(2)) // Speed

		if err != nil {
			fmt.Println("Error while reading GPS Element")
			break
		}

		elements[i] = Record{
			imei,
			Location{"Point",
				[]float64{longitude, latitude}},
			timestamp,
			angle,
			speed}

		// IO Events Elements

		reader.Next(1) // ioEventID
		reader.Next(1) // total Elements

		stage := 1
		for stage <= 4 {
			stageElements, err := streamToInt8(reader.Next(1))
			if err != nil {
				break
			}

			var j int8 = 0
			for j < stageElements {
				reader.Next(1) // elementID

				switch stage {
				case 1: // One byte IO Elements
					_, err = streamToInt8(reader.Next(1))
				case 2: // Two byte IO Elements
					_, err = streamToInt16(reader.Next(2))
				case 3: // Four byte IO Elements
					_, err = streamToInt32(reader.Next(4))
				case 4: // Eigth byte IO Elements
					_, err = streamToInt64(reader.Next(8))
				}
				j++
			}
			stage++
		}

		if err != nil {
			fmt.Println("Error while reading IO Elements")
			break
		}

		fmt.Println("Timestamp:", timestamp)
		fmt.Println("Longitude:", longitude, "Latitude:", latitude)

		i++
	}

	// Once finished with the records we read the Record Number and the CRC

	_, err = streamToInt8(reader.Next(1))  // Number of Records
	_, err = streamToInt32(reader.Next(4)) // CRC

	return
}

func (s *TeltonikaServer) Stop() {
	s.wg.Wait()
	s.quitChan <- Empty{}
	s.log.Info("stop server")
}
