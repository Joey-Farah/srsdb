package storage

import (
	"encoding/binary"
	"fmt"
)

// example
// type rawPlayer struct {
// 	Character int    `json:"character"`
// 	Port      string `json:"port"`
// }

// Records grow from the front of the page right (contents)
// Slots grow from the end of the page left (table of contents)

// a []byte is a slice, it has the below values
//pointer   → where the bytes live
// length    → how many bytes (4096)
// capacity  → how much room is available

type Slot struct {
	Offset uint16
	Length uint16
}

const slotSize = 4
const headerSize = 2

func putSlot(page []byte, position int, slot Slot) {
	binary.LittleEndian.PutUint16(page[position:], slot.Offset)
	binary.LittleEndian.PutUint16(page[position+2:], slot.Length)
}

func getSlot(page []byte, position int) Slot {
	offset := binary.LittleEndian.Uint16(page[position:])
	length := binary.LittleEndian.Uint16(page[position+2:])

	return Slot{
		Offset: offset,
		Length: length,
	}
}

func putNumSlots(page []byte, numSlots uint16) {
	binary.LittleEndian.PutUint16(page, numSlots)
}

func getNumSlots(page []byte) uint16 {
	numSlot := binary.LittleEndian.Uint16(page)
	return numSlot
}

func slotPosition(numSlots uint16) int {
	// Size of page in bytes subtracted by the number of slots plus the slot we're adding multiplied by number of bytes in a slot
	return PageSize - ((int(numSlots) + 1) * slotSize)

}

func recordStart(page []byte) int {
	numSlots := getNumSlots(page)
	if numSlots == 0 {
		return headerSize
	}
	lastSlot := getSlot(page, slotPosition(numSlots-1))
	newRecordPosition := int(lastSlot.Offset) + int(lastSlot.Length)

	return newRecordPosition
}

// Calling recordStart gives us the position of the new record, so we shouldn't need to take it in
// Error handling needs to make sure the location its writing bytes to doesn't already contain anything written
// if the size of the bytes being written is greater than the size of the remaining unwritten bytes, then throw an error
func insertRecord(page []byte, newRecord []byte) (newSlotNumber uint16, err error) {
	//grabs index of the new slot (just the number of slots on the page)
	newSlotNumber = getNumSlots(page)

	newRecordStart := recordStart(page)
	newRecordEndPosition := newRecordStart + len(newRecord)
	newSlotPosition := slotPosition(newSlotNumber)

	// end = first byte AFTER the record, so end == slot position is an exact fit; only end > slot position overlaps
	if newRecordEndPosition > int(newSlotPosition) {
		return newSlotNumber, fmt.Errorf("Page out of space.")
	}
	return newSlotNumber, err
}
