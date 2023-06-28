package simulator

import (
	"github.com/openfms/teltonika-device/parser"
	"math/rand"
	"time"
)

func generateRandomAVLData() *parser.AVLData {
	avlData := &parser.AVLData{
		Timestamp:  uint64(time.Now().Unix()),
		Priority:   getRandomPacketPriority(),
		Longitude:  getRandomFloat64(-180, 180),
		Latitude:   getRandomFloat64(-90, 90),
		Altitude:   int16(getRandomInt(-1000, 1000)),
		Angle:      uint16(getRandomInt(0, 360)),
		Satellites: uint8(getRandomInt(0, 12)),
		Speed:      uint16(getRandomInt(0, 200)),
		EventID:    uint16(getRandomInt(0, 100)),
		IOElements: generateRandomIOElements(),
	}
	return avlData
}

func getRandomPacketPriority() parser.PacketPriority {
	priorities := []parser.PacketPriority{
		parser.PriorityLow,
		parser.PriorityHigh,
		parser.PriorityPanic,
	}

	return priorities[getRandomInt(0, len(priorities))]
}

func generateRandomIOElements() []*parser.IOElement {
	numIOElements := getRandomInt(1, 5)
	elements := make([]*parser.IOElement, numIOElements)

	for i := 0; i < numIOElements; i++ {
		ioElement := &parser.IOElement{
			ID:    uint16(getRandomInt(1, 100)),
			Value: getRandomValue(),
		}
		elements[i] = ioElement
	}

	return elements
}

func getRandomValue() any {
	// Generate a random value of any type (e.g., int, float64, string)
	value := uint32(getRandomInt(1, 100))

	// Add more cases here if you want to generate values of different types
	return value
}

func getRandomFloat64(min, max float64) float64 {
	randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))
	return min + randomizer.Float64()*(max-min)
}

func getRandomInt(min, max int) int {
	randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))
	return min + randomizer.Intn(max-min+1)
}
