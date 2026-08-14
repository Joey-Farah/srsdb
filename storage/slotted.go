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
