package storage

import (
	"encoding/binary"
)

// example
// type rawPlayer struct {
// 	Character int    `json:"character"`
// 	Port      string `json:"port"`
// }

type Slot struct {
	Offset uint16
	Length uint16
}

const slotSize = 4
const headerSize = 2

func putSlot(pageBuffer []byte, position int, slot Slot) {
	binary.LittleEndian.PutUint16(pageBuffer[position:], slot.Offset)
	binary.LittleEndian.PutUint16(pageBuffer[position+2:], slot.Length)
}

func getSlot(pageBuffer []byte, position int) Slot {
	offset := binary.LittleEndian.Uint16(pageBuffer[position:])
	length := binary.LittleEndian.Uint16(pageBuffer[position+2:])

	return Slot{
		Offset: offset,
		Length: length,
	}
}

func putNumSlots(pageBuffer []byte, numSlots uint16) {
	binary.LittleEndian.PutUint16(pageBuffer, numSlots)
}

func getNumSlots(pageBuffer []byte) uint16 {
	numSlot := binary.LittleEndian.Uint16(pageBuffer)
	return numSlot
}

func slotPosition(numSlots uint16) int {
	// Size of page in bytes subtracted by the number of slots plus the slot we're adding multiplied by number of bytes in a slot
	return PageSize - ((int(numSlots) + 1) * slotSize)

}

func recordStart(pageBuffer []byte) int {
	numSlots := getNumSlots(pageBuffer)
	if numSlots == 0 {
		return headerSize
	}
	lastSlot := getSlot(pageBuffer, slotPosition(numSlots-1))
	newRecordPosition := int(lastSlot.Offset) + int(lastSlot.Length)

	return newRecordPosition
}

func insertRecord() int {
	return -1
}
