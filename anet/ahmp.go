package anet

import (
	"bytes"
	"io"
	"strconv"

	"github.com/quic-go/quic-go"
)

type AHMPError struct {
	msg string
}

func (e *AHMPError) Error() string {
	return e.msg
}

func NewAHMPError(msg string) *AHMPError {
	result := new(AHMPError)
	result.msg = msg
	return result
}

type AHMPExit struct {
	exitcode error
}

type AHMPRaw_ID struct {
	name   []byte
	pubkey []byte
}

type AHMPRaw_JN struct {
	path []byte
}

type AHMPRaw_JOK struct {
	path  []byte
	world []byte
}

type AHMPRaw_JDN struct {
	path    []byte
	status  int
	message []byte
}

type AHMPRaw_JNI struct {
	world_uuid []byte
	address    []byte
}

type AHMPRaw_MEM struct {
	world_uuid []byte
}

type AHMPRaw_SNB struct {
	world_uuid   []byte
	members_hash []byte
}

type AHMPRaw_CRR struct {
	world_uuid   []byte
	missing_hash []byte
}

type AHMPRaw_RST struct {
	world_uuid []byte
}

type AHMPParser struct {
	buffer bytes.Buffer
}

type quicReader struct {
	stream quic.Stream
}

func (r quicReader) Read(p []byte) (int, error) {
	n, err := r.stream.Read(p)
	if err != nil {
		return n, err
	}
	return n, io.EOF
}

func _Split2(b []byte) ([]byte, []byte, bool) {
	s := bytes.IndexByte(b, ' ')
	if s == -1 {
		return nil, nil, false
	}
	return b[:s], b[s+1:], true
}

func _Split3(b []byte) ([]byte, []byte, []byte, bool) {
	i, j, ok := _Split2(b)
	if !ok {
		return nil, nil, nil, false
	}
	jk, jq, ok := _Split2(j)
	return i, jk, jq, ok
}

func (p *AHMPParser) Read(stream quic.Stream) (any, error) {
	reader := quicReader{stream: stream}
	GetLine := func() ([]byte, error) {
		for {
			line, err := p.buffer.ReadBytes('\n')
			if err == io.EOF {
				if p.buffer.Len() > 128 {
					return nil, NewAHMPError("ahmp too long line")
				}
				_, err = p.buffer.ReadFrom(reader)
				if err != nil {
					return nil, err
				}
				continue
			}
			if err != nil {
				return nil, err
			}
			return line[:len(line)-1], nil
		}
	}
	GetBody := func(content_length int) ([]byte, error) {
		for p.buffer.Len() < content_length {
			_, err := p.buffer.ReadFrom(reader)
			if err != nil {
				return nil, err
			}
		}
		return p.buffer.Next(content_length), nil
	}
	ContentLengthHeaderOnlyBodyParse := func() ([]byte, error) {
		headerline, err := GetLine()
		if err != nil {
			return nil, err
		}
		body_len_str, ok := bytes.CutPrefix(headerline, []byte("Content-Length: "))
		if !ok {
			return nil, err
		}
		body_len, err := strconv.Atoi(string(body_len_str))
		if err != nil {
			return nil, err
		}

		headerline, err = GetLine()
		if err != nil {
			return nil, err
		}
		if len(headerline) != 0 {
			return nil, NewAHMPError("unsupported headers")
		}

		return GetBody(body_len)
	}
	NoBodyFinish := func() error {
		headerline, err := GetLine()
		if err != nil {
			return err
		}
		if len(headerline) != 0 {
			return NewAHMPError("unsupported headers")
		}
		return nil
	}

	line, err := GetLine()
	if err != nil {
		return nil, err
	}

	pos := bytes.IndexByte(line, ' ')
	if !bytes.Equal(line[:pos], []byte("AHMP/1.0")) {
		return nil, NewAHMPError("unknown AHMP subprotocol: " + string(line[:pos]))
	}
	line = line[pos+1:]

	pos = bytes.IndexByte(line, ' ')
	if pos == -1 {
		return nil, NewAHMPError("unknown AHMP method: " + string(line))
	}

	method := string(line[:pos])
	args := line[pos+1:]
	var ok bool
	switch method {
	case "ID":
		var parsed AHMPRaw_ID
		parsed.name = args
		parsed.pubkey, err = ContentLengthHeaderOnlyBodyParse()
		return parsed, err
	case "JN":
		return AHMPRaw_JN{path: args}, NoBodyFinish()
	case "JOK":
		var parsed AHMPRaw_JOK
		parsed.path = args
		parsed.world, err = ContentLengthHeaderOnlyBodyParse()
		return parsed, err
	case "JDN":
		var parsed AHMPRaw_JDN
		var a2 []byte
		parsed.path, a2, parsed.message, ok = _Split3(args)
		if !ok {
			return parsed, NewAHMPError("malformed JDN message")
		}
		parsed.status, err = strconv.Atoi(string(a2))
		if err != nil {
			return parsed, NewAHMPError("malformed JDN message")
		}
		return parsed, NoBodyFinish()
	case "JNI":
		var parsed AHMPRaw_JNI
		parsed.world_uuid, parsed.address, ok = _Split2(args)
		if !ok {
			return parsed, NewAHMPError("malformed JNI message")
		}
		return parsed, NoBodyFinish()
	case "MEM":
		return AHMPRaw_MEM{world_uuid: args}, NoBodyFinish()
	case "SNB":
		var parsed AHMPRaw_SNB
		parsed.world_uuid = args
		parsed.members_hash, err = ContentLengthHeaderOnlyBodyParse()
		return parsed, err
	case "CRR":
		var parsed AHMPRaw_CRR
		parsed.world_uuid, parsed.missing_hash, ok = _Split2(args)
		if !ok {
			return parsed, NewAHMPError("malformed CRR message")
		}
		return parsed, NoBodyFinish()
	case "RST":
		return AHMPRaw_RST{world_uuid: args}, NoBodyFinish()
	default:
		return nil, NewAHMPError("unknown AHMP method: " + method)
	}
}
