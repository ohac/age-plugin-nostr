package pngpayload

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
)

var signature = []byte{137, 80, 78, 71, 13, 10, 26, 10}

var payloadChunk = [4]byte{'n', 'a', 'G', 'e'}

func IsPNG(data []byte) bool {
	return len(data) >= len(signature) && string(data[:len(signature)]) == string(signature)
}

type chunk struct {
	start int
	end   int
	type_ [4]byte
}

func Embed(image, payload []byte) ([]byte, error) {
	if uint64(len(payload)) > 0x7fffffff {
		return nil, errors.New("encrypted payload is too large for a PNG chunk")
	}
	chunks, err := parse(image)
	if err != nil {
		return nil, err
	}

	base := make([]byte, 0, len(image))
	base = append(base, signature...)
	for _, item := range chunks {
		if item.type_ == payloadChunk {
			continue
		}
		base = append(base, image[item.start:item.end]...)
	}
	chunks, err = parse(base)
	if err != nil {
		return nil, err
	}
	insertAt := chunks[len(chunks)-1].start
	encodedChunk := makeChunk(payloadChunk, payload)

	result := make([]byte, 0, len(base)+len(encodedChunk))
	result = append(result, base[:insertAt]...)
	result = append(result, encodedChunk...)
	result = append(result, base[insertAt:]...)
	return result, nil
}

func Extract(image []byte) ([]byte, error) {
	chunks, err := parse(image)
	if err != nil {
		return nil, err
	}
	var found []byte
	for _, item := range chunks {
		if item.type_ != payloadChunk {
			continue
		}
		if found != nil {
			return nil, errors.New("PNG contains multiple nostr-age payload chunks")
		}
		length := int(binary.BigEndian.Uint32(image[item.start : item.start+4]))
		found = image[item.start+8 : item.start+8+length]
	}
	if found == nil {
		return nil, errors.New("PNG does not contain a nostr-age payload")
	}
	return append([]byte(nil), found...), nil
}

func parse(image []byte) ([]chunk, error) {
	if len(image) < len(signature) || string(image[:len(signature)]) != string(signature) {
		return nil, errors.New("input is not a PNG file")
	}
	var chunks []chunk
	hasImageData := false
	for offset := len(signature); ; {
		if len(image)-offset < 12 {
			return nil, errors.New("truncated PNG chunk")
		}
		length := uint64(binary.BigEndian.Uint32(image[offset : offset+4]))
		if length > 0x7fffffff {
			return nil, errors.New("PNG chunk exceeds the maximum permitted size")
		}
		end := uint64(offset) + 12 + length
		if end > uint64(len(image)) {
			return nil, errors.New("truncated PNG chunk data")
		}
		var name [4]byte
		copy(name[:], image[offset+4:offset+8])
		chunkDataEnd := offset + 8 + int(length)
		wantCRC := binary.BigEndian.Uint32(image[chunkDataEnd : chunkDataEnd+4])
		if crc32.ChecksumIEEE(image[offset+4:chunkDataEnd]) != wantCRC {
			return nil, fmt.Errorf("PNG chunk %.4s has an invalid CRC", name)
		}
		item := chunk{start: offset, end: int(end), type_: name}
		chunks = append(chunks, item)
		if len(chunks) == 1 && (name != [4]byte{'I', 'H', 'D', 'R'} || length != 13) {
			return nil, errors.New("PNG does not start with a valid IHDR chunk")
		}
		if name == [4]byte{'I', 'E', 'N', 'D'} {
			if length != 0 || !hasImageData || int(end) != len(image) {
				return nil, errors.New("invalid PNG IEND chunk or trailing data")
			}
			return chunks, nil
		}
		if name == [4]byte{'I', 'D', 'A', 'T'} {
			hasImageData = true
		}
		offset = int(end)
	}
}

func makeChunk(name [4]byte, data []byte) []byte {
	result := make([]byte, 12+len(data))
	binary.BigEndian.PutUint32(result[:4], uint32(len(data)))
	copy(result[4:8], name[:])
	copy(result[8:8+len(data)], data)
	binary.BigEndian.PutUint32(result[8+len(data):], crc32.ChecksumIEEE(result[4:8+len(data)]))
	return result
}
