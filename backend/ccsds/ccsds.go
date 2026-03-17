// Package ccsds implements the CCSDS Space Packet Protocol (CCSDS 133.0-B-2).
//
//	┌─────────────────────── Primary Header (6 bytes) ───────────────────────┐
//	│ Version(3) | Type(1) | SecHdr(1) | APID(11) | SeqFlags(2) | SeqCnt(14) | DataLen(16) │
//	└────────────────────────────────────────────────────────────────────────┘
//	│                        Data Field (variable)                          │
//	└────────────────────────────────────────────────────────────────────────┘
//	│                        CRC-16 (2 bytes)                               │
//	└────────────────────────────────────────────────────────────────────────┘
package ccsds

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Packet types
const (
	TypeTelemetry = 0
	TypeCommand   = 1
)

// Sequence flags
const (
	SeqContinuation = 0
	SeqFirst        = 1
	SeqLast         = 2
	SeqStandalone   = 3
)

// Well-known APIDs for SpaceDrone subsystems
const (
	APIDMissionControl  = 0x01
	APIDGNC             = 0x02
	APIDPowerManagement = 0x03
	APIDCommunications  = 0x04
	APIDObjectDetection = 0x05
	APIDHealthMonitor   = 0x06
)

// PrimaryHeader represents the 6-byte CCSDS primary header.
type PrimaryHeader struct {
	Version      uint8  // 3 bits — always 0
	PacketType   uint8  // 1 bit — 0=telemetry, 1=command
	SecHdrFlag   bool   // 1 bit
	APID         uint16 // 11 bits — application process identifier
	SeqFlags     uint8  // 2 bits
	SeqCount     uint16 // 14 bits
	DataLength   uint16 // 16 bits — (data field length) - 1
}

// Packet represents a complete CCSDS space packet.
type Packet struct {
	Header  PrimaryHeader
	Data    []byte
	CRC     uint16
}

// Encode serializes a CCSDS packet into bytes.
func (p *Packet) Encode() []byte {
	totalLen := 6 + len(p.Data) + 2 // header + data + CRC
	buf := make([]byte, totalLen)

	// Word 0: Version(3) | Type(1) | SecHdrFlag(1) | APID(11)
	word0 := uint16(p.Header.Version&0x07) << 13
	word0 |= uint16(p.Header.PacketType&0x01) << 12
	if p.Header.SecHdrFlag {
		word0 |= 1 << 11
	}
	word0 |= p.Header.APID & 0x07FF
	binary.BigEndian.PutUint16(buf[0:2], word0)

	// Word 1: SeqFlags(2) | SeqCount(14)
	word1 := uint16(p.Header.SeqFlags&0x03) << 14
	word1 |= p.Header.SeqCount & 0x3FFF
	binary.BigEndian.PutUint16(buf[2:4], word1)

	// Word 2: DataLength (number of octets in data field minus 1)
	p.Header.DataLength = uint16(len(p.Data) - 1)
	binary.BigEndian.PutUint16(buf[4:6], p.Header.DataLength)

	// Data field
	copy(buf[6:6+len(p.Data)], p.Data)

	// CRC-16 over header + data
	crc := CRC16(buf[:6+len(p.Data)])
	p.CRC = crc
	binary.BigEndian.PutUint16(buf[6+len(p.Data):], crc)

	return buf
}

// Decode parses raw bytes into a CCSDS packet.
func Decode(raw []byte) (*Packet, error) {
	if len(raw) < 9 { // 6 header + 1 data min + 2 CRC
		return nil, errors.New("packet too short: minimum 9 bytes required")
	}

	p := &Packet{}

	// Parse word 0
	word0 := binary.BigEndian.Uint16(raw[0:2])
	p.Header.Version = uint8((word0 >> 13) & 0x07)
	p.Header.PacketType = uint8((word0 >> 12) & 0x01)
	p.Header.SecHdrFlag = (word0>>11)&0x01 == 1
	p.Header.APID = word0 & 0x07FF

	// Parse word 1
	word1 := binary.BigEndian.Uint16(raw[2:4])
	p.Header.SeqFlags = uint8((word1 >> 14) & 0x03)
	p.Header.SeqCount = word1 & 0x3FFF

	// Parse word 2
	p.Header.DataLength = binary.BigEndian.Uint16(raw[4:6])
	dataLen := int(p.Header.DataLength) + 1

	if len(raw) < 6+dataLen+2 {
		return nil, fmt.Errorf("packet truncated: need %d bytes, have %d", 6+dataLen+2, len(raw))
	}

	// Data field
	p.Data = make([]byte, dataLen)
	copy(p.Data, raw[6:6+dataLen])

	// CRC verification
	p.CRC = binary.BigEndian.Uint16(raw[6+dataLen : 6+dataLen+2])
	computed := CRC16(raw[:6+dataLen])
	if p.CRC != computed {
		return nil, fmt.Errorf("CRC mismatch: got 0x%04X, computed 0x%04X", p.CRC, computed)
	}

	return p, nil
}

// CRC16 computes a CRC-16/CCITT-FALSE over the given data.
// Polynomial: 0x1021, Init: 0xFFFF
func CRC16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// NewTelemetryPacket creates a CCSDS telemetry packet for the given APID.
func NewTelemetryPacket(apid uint16, seqCount uint16, data []byte) *Packet {
	return &Packet{
		Header: PrimaryHeader{
			Version:    0,
			PacketType: TypeTelemetry,
			SecHdrFlag: false,
			APID:       apid,
			SeqFlags:   SeqStandalone,
			SeqCount:   seqCount,
		},
		Data: data,
	}
}

// NewCommandPacket creates a CCSDS command packet for the given APID.
func NewCommandPacket(apid uint16, seqCount uint16, data []byte) *Packet {
	return &Packet{
		Header: PrimaryHeader{
			Version:    0,
			PacketType: TypeCommand,
			SecHdrFlag: false,
			APID:       apid,
			SeqFlags:   SeqStandalone,
			SeqCount:   seqCount,
		},
		Data: data,
	}
}

// APIDName returns a human-readable name for known APIDs.
func APIDName(apid uint16) string {
	switch apid {
	case APIDMissionControl:
		return "MissionControl"
	case APIDGNC:
		return "GNC"
	case APIDPowerManagement:
		return "PowerManagement"
	case APIDCommunications:
		return "Communications"
	case APIDObjectDetection:
		return "ObjectDetection"
	case APIDHealthMonitor:
		return "HealthMonitor"
	default:
		return fmt.Sprintf("APID-%d", apid)
	}
}
