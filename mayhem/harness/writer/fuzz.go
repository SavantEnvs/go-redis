package writer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"

	"github.com/redis/go-redis/v9/internal/proto"
)

// Fuzz encodes one argument with proto.Writer.WriteArg, reads it back with
// proto.Reader and checks the RESP bulk string against the expected text.
// Oracle: a nil pointer to a numeric/bool type encodes exactly like the zero
// value of that type, and a non-nil one encodes its value.
func Fuzz(data []byte) int {
	if len(data) < 2 {
		return -1
	}
	kind, isNil, rest := data[0]%24, data[1]&1 == 1, data[2:]

	var buf [8]byte
	copy(buf[:], rest)
	u := binary.LittleEndian.Uint64(buf[:])

	var arg interface{}
	var want string

	switch kind {
	case 0:
		var p *int
		v := int(int64(u))
		if !isNil {
			p, want = &v, strconv.FormatInt(int64(v), 10)
		} else {
			want = "0"
		}
		arg = p
	case 1:
		var p *int8
		v := int8(u)
		if !isNil {
			p, want = &v, strconv.FormatInt(int64(v), 10)
		} else {
			want = "0"
		}
		arg = p
	case 2:
		var p *int16
		v := int16(u)
		if !isNil {
			p, want = &v, strconv.FormatInt(int64(v), 10)
		} else {
			want = "0"
		}
		arg = p
	case 3:
		var p *int32
		v := int32(u)
		if !isNil {
			p, want = &v, strconv.FormatInt(int64(v), 10)
		} else {
			want = "0"
		}
		arg = p
	case 4:
		var p *int64
		v := int64(u)
		if !isNil {
			p, want = &v, strconv.FormatInt(v, 10)
		} else {
			want = "0"
		}
		arg = p
	case 5:
		var p *uint
		v := uint(u)
		if !isNil {
			p, want = &v, strconv.FormatUint(uint64(v), 10)
		} else {
			want = "0"
		}
		arg = p
	case 6:
		var p *uint8
		v := uint8(u)
		if !isNil {
			p, want = &v, strconv.FormatUint(uint64(v), 10)
		} else {
			want = "0"
		}
		arg = p
	case 7:
		var p *uint16
		v := uint16(u)
		if !isNil {
			p, want = &v, strconv.FormatUint(uint64(v), 10)
		} else {
			want = "0"
		}
		arg = p
	case 8:
		var p *uint32
		v := uint32(u)
		if !isNil {
			p, want = &v, strconv.FormatUint(uint64(v), 10)
		} else {
			want = "0"
		}
		arg = p
	case 9:
		var p *uint64
		v := u
		if !isNil {
			p, want = &v, strconv.FormatUint(v, 10)
		} else {
			want = "0"
		}
		arg = p
	case 10:
		var p *float32
		v := math.Float32frombits(uint32(u))
		if !isNil {
			p, want = &v, strconv.FormatFloat(float64(v), 'f', -1, 64)
		} else {
			want = "0"
		}
		arg = p
	case 11:
		var p *float64
		v := math.Float64frombits(u)
		if !isNil {
			p, want = &v, strconv.FormatFloat(v, 'f', -1, 64)
		} else {
			want = "0"
		}
		arg = p
	case 12:
		var p *bool
		v := u&1 == 1
		if !isNil {
			p = &v
			want = "0"
			if v {
				want = "1"
			}
		} else {
			want = "0"
		}
		arg = p
	case 13:
		arg, want = int(int64(u)), strconv.FormatInt(int64(int(int64(u))), 10)
	case 14:
		arg, want = int8(u), strconv.FormatInt(int64(int8(u)), 10)
	case 15:
		arg, want = int64(u), strconv.FormatInt(int64(u), 10)
	case 16:
		arg, want = uint8(u), strconv.FormatUint(uint64(uint8(u)), 10)
	case 17:
		arg, want = uint16(u), strconv.FormatUint(uint64(uint16(u)), 10)
	case 18:
		arg, want = uint32(u), strconv.FormatUint(uint64(uint32(u)), 10)
	case 19:
		arg, want = u, strconv.FormatUint(u, 10)
	case 20:
		arg, want = math.Float64frombits(u), strconv.FormatFloat(math.Float64frombits(u), 'f', -1, 64)
	case 21:
		s := string(rest)
		arg, want = s, s
	case 22:
		arg, want = append([]byte(nil), rest...), string(rest)
	case 23:
		b := u&1 == 1
		arg, want = b, "0"
		if b {
			want = "1"
		}
	}

	var out bytes.Buffer
	if err := proto.NewWriter(&out).WriteArg(arg); err != nil {
		panic(err)
	}
	got, err := proto.NewReader(bytes.NewReader(out.Bytes())).ReadReply()
	if err != nil {
		panic(fmt.Sprintf("WriteArg(%T) wrote %q, unreadable: %v", arg, out.Bytes(), err))
	}
	if s, ok := got.(string); !ok || s != want {
		panic(fmt.Sprintf("WriteArg(%T nil=%v) wrote %q, read back %#v, want %q", arg, isNil, out.Bytes(), got, want))
	}
	return 0
}
