// This package implements WAVE file encoding and decoding.
package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	// ASCII string "RIFF"
	ID_RIFF uint32 = 0x52494646
	// ASCII string "WAVE"
	ID_WAVE uint32 = 0x57415645
	// ASCII string "fmt "
	ID_fmt uint32 = 0x666d7420
	// ASCII string "data"
	ID_data uint32 = 0x64617461
)

type RIFF struct {
	Header
	FmtChunk
	DataChunk
}

// Chunk header/descriptor.
//
// Id is ASCII of "RIFF" or 0x52494646.
//
// Size is the size of entire file excluding Chunk Id and Chunk Size.
// 4 + (8 + FmtChunkSize) + (8 + DataChunkSize). Equivalent to the
// size of the file from this point onwards.
//
// Format is ASCII of "WAVE" or 0x57415645
type Header struct {
	Id     uint32 //  4 bytes big endian
	Size   uint32 //  4 bytes little endian
	Format uint32 //  4 bytes big endian
}

// fmt sub-chunk.
type FmtChunk struct {
	ID          uint32 // 4 bytes big endian
	Size        uint32 // 4 bytes little endian
	AudioFormat uint16 // 2 bytes little endian
	Channel     uint16 // 2 bytes little endian
	// Sample per second
	SampleRate    uint32 // 4 bytes little endian
	ByteRate      uint32 // 4 bytes little endian
	BlockAlign    uint16 // 2 bytes little endian
	BitsPerSample uint16 // 2 bytes little endian
}

type DataChunk struct {
	ID   uint32 // 4 bytes big endian
	Size uint32 // 4 bytes little endian
	Data io.Reader
}

func main() {
	file, err := os.Open("test.wav")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()

	// h, err := ParseHeader(file)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Printf("%x\n", h)
	// fm, err := ParseFmtChunk(file)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Printf("%x\n", fm)
	// dat, err := ParseDataChunk(file)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Printf("%x\n", dat)

	riffwave := RIFF{}
	err = riffwave.Parse(file)
	fmt.Printf("%v\n", riffwave)
	riffwave.ParseData(file)

}

func (f *RIFF) Parse(r io.Reader) (err error) {
	f.Header, err = ParseHeader(r)
	if err != nil {
		return err
	}
	f.FmtChunk, err = ParseFmtChunk(r)
	if err != nil {
		return err
	}
	f.DataChunk, err = ParseDataChunk(r)
	if err != nil {
		return err
	}
	return err
}

func ParseHeader(r io.Reader) (Header, error) {
	header := Header{}
	buf := make([]byte, 12)
	_, err := r.Read(buf)
	if err != nil {
		return header, err
	}
	header.Id = binary.BigEndian.Uint32(buf[0:4])
	if header.Id != ID_RIFF {
		return header, errors.New("not a valid RIFF file")
	}
	// TODO: check for file size
	header.Size = binary.LittleEndian.Uint32(buf[4:8])
	header.Format = binary.BigEndian.Uint32(buf[8:12])
	if header.Format != ID_WAVE {
		return header, errors.New("not a valid WAVE file")
	}
	return header, err
}

// TODO: sanity check
func ParseFmtChunk(r io.Reader) (FmtChunk, error) {
	fm := FmtChunk{}
	buf := make([]byte, 24)
	_, err := r.Read(buf)
	if err != nil {
		return fm, err
	}
	fm.ID = binary.BigEndian.Uint32(buf[0:4])
	if fm.ID != ID_fmt {
		return fm, errors.New("not a valid fmt chunk")
	}
	fm.Size = binary.LittleEndian.Uint32(buf[4:8])
	fm.AudioFormat = binary.LittleEndian.Uint16(buf[8:10])
	fm.Channel = binary.LittleEndian.Uint16(buf[10:12])
	fm.SampleRate = binary.LittleEndian.Uint32(buf[12:16])
	fm.ByteRate = binary.LittleEndian.Uint32(buf[16:20])
	fm.BlockAlign = binary.LittleEndian.Uint16(buf[20:22])
	fm.BitsPerSample = binary.LittleEndian.Uint16(buf[22:24])
	return fm, err
}

// TODO: sanity check
func ParseDataChunk(r io.Reader) (DataChunk, error) {
	dataChunk := DataChunk{}
	buf := make([]byte, 8)
	_, err := r.Read(buf)
	if err != nil {
		return dataChunk, err
	}
	dataChunk.ID = binary.BigEndian.Uint32(buf[0:4])
	if dataChunk.ID != ID_data {
		return dataChunk, errors.New("not a valid data chunk")
	}
	dataChunk.Size = binary.LittleEndian.Uint32(buf[4:8])
	dataChunk.Data = r
	return dataChunk, err
}

type ChannelData []int

func (f *RIFF) ParseData(r io.Reader) error {
	var err error
	sampleCount := int(f.DataChunk.Size) / int(f.FmtChunk.Channel) / int(f.FmtChunk.BitsPerSample/8)
	data := make([]ChannelData, f.FmtChunk.Channel)

	for i := range data {
		chData := make([]int, sampleCount)
		data[i] = chData
	}

	// Enough buffer to fit Uint32 for 32 bits per sample
	buf := make([]byte, 4)
	for i := 0; i < sampleCount; i++ {
		for j := 0; j < len(data); j++ {
			_, err := r.Read(buf)
			if err != nil && err != io.EOF {
				return errors.New("fail to parse sample data")
			}
			switch f.FmtChunk.BitsPerSample {
			case 8:
				data[j][i] = int(int8(buf[0]))
			case 16:
				data[j][i] = int(int16(binary.LittleEndian.Uint16(buf)))
			case 32:
				data[j][i] = int(int32(binary.LittleEndian.Uint32(buf)))
			default:
				return errors.New("unsupported bits per sample")

			}

		}
	}

	fmt.Println(data[0][0:100])
	return err

	// var sampleparser func([]byte) uint16

	// switch f.FmtChunk.BitsPerSample {
	// case 8:
	// 	sampleparser = binary.LittleEndian.Uint16
	// case 16:
	// 	sampleparser = binary.LittleEndian.Uint16
	// case 32:
	// 	sampleparser = binary.LittleEndian.Uint16
	// default:
	// 	return errors.New("Unsupported bits per sample")

	// }

	// buf := make([]byte, 2)
	// for i := 0; i < samples; i++ {
	// 	for j := 0; j < int(f.FmtChunk.Channel); j++ {
	// 		channelData[i][j] = int(sampleparser(r))
	// 	}
	// }
	// fmt.Println(channelData[0][0])
}
