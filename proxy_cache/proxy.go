package proxy_cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"
)

type Proxy struct {
	Addr        string
	lastCheck   time.Time
	failCounter uint
}

func (p *Proxy) String() string {
	return fmt.Sprintf(
		"%v %v %d", p.Addr, p.lastCheck.Format(time.RFC3339),
		p.failCounter,
	)
}

func (p *Proxy) GobDecode(b []byte) error {
	buffer := bytes.NewReader(b)
	decoder := gob.NewDecoder(buffer)
	err := decoder.Decode(&p.Addr)
	if err == nil {
		err = decoder.Decode(&p.lastCheck)
	}
	if err == nil {
		err = decoder.Decode(&p.failCounter)
	}
	return err
}

func (p *Proxy) GobEncode() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	encoder := gob.NewEncoder(buffer)
	if err := encoder.Encode(p.Addr); err != nil {
		return nil, err
	}
	if err := encoder.Encode(p.lastCheck); err != nil {
		return nil, err
	}
	if err := encoder.Encode(p.failCounter); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
