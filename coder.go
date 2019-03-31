package stream

import (
	"io"
	"encoding/binary"
	"encoding/json"
)

type Encoder interface {
	Encode(interface{}, int) error
}

type Decoder interface {
	Decode(interface{}) error
}

type msgDecoder struct {
	r io.Reader
}

func NewMsgDecoder(r io.Reader) Decoder {
	return &msgDecoder{r}
}

func (dec *msgDecoder) Decode(v interface{}) error {
	var l uint64
	if err := binary.Read(dec.r, binary.BigEndian, &l); err != nil {
		return err
	}

	buf := make([]byte, int(l))
	if _, err := io.ReadFull(dec.r, buf); err != nil {
		return err
	}

	return json.Unmarshal(buf, v)
}

type msgEncoder struct {
	w io.Writer
}

func NewMsgEncoder(w io.Writer) Encoder {
	return &msgEncoder{
		w,
	}
}

func (enc *msgEncoder) Encode(m interface{}, size int) error {
	v, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := binary.Write(enc.w, binary.BigEndian, uint64(size)); err != nil {
		return nil
	}

	_, err = enc.w.Write(v)
	return err
}
