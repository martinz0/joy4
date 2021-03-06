// Packege pubsub implements publisher-subscribers model used in multi-channel streaming.
package pubsub

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/martinz0/joy4/av"
	"github.com/martinz0/joy4/av/pktque"
	"github.com/martinz0/joy4/codec/aacparser"
	"github.com/martinz0/joy4/codec/h264parser"
)

//        time
// ----------------->
//
// V-A-V-V-A-V-V-A-V-V
// |                 |
// 0        5        10
// head             tail
// oldest          latest
//

// One publisher and multiple subscribers thread-safe packet buffer queue.
type Queue struct {
	buf                      *pktque.Buf
	head, tail               int
	lock                     *sync.RWMutex
	cond                     *sync.Cond
	curgopcount, maxgopcount int
	streams                  []av.CodecData
	videoidx                 int
	audioidx                 int
	closed                   bool

	jumped bool
}

func NewQueue() *Queue {
	q := &Queue{}
	q.buf = pktque.NewBuf()
	q.maxgopcount = 2
	q.lock = &sync.RWMutex{}
	q.cond = sync.NewCond(q.lock.RLocker())
	q.videoidx = -1
	q.audioidx = -1
	return q
}

func (self *Queue) SetMaxGopCount(n int) {
	self.lock.Lock()
	self.maxgopcount = n
	self.lock.Unlock()
	return
}

func (self *Queue) WriteHeader(streams []av.CodecData) error {
	self.lock.Lock()

	self.streams = streams
	for i, stream := range streams {
		if stream.Type().IsVideo() {
			self.videoidx = i
		} else if stream.Type().IsAudio() {
			self.audioidx = i
		}
	}
	self.cond.Broadcast()

	self.lock.Unlock()

	return nil
}

func (self *Queue) WriteTrailer() error {
	return nil
}

// After Close() called, all QueueCursor's ReadPacket will return io.EOF.
func (self *Queue) Close() (err error) {
	self.lock.Lock()

	self.closed = true
	self.cond.Broadcast()

	self.lock.Unlock()
	return
}

// Put packet into buffer, old packets will be discared.
func (self *Queue) WritePacket(pkt av.Packet) (err error) {
	self.lock.Lock()

	if pkt.IsKeyFrame && pkt.IsVideo {
		// 拿到I帧, 清空前面所有帧
		// if self.buf.Count > 0 {
		// }
		// println("drop frames: ", self.buf.Count)
		for self.buf.Count > 0 {
			poped := self.buf.Pop()
			if poped.IsSeqHDR {
				self.jumped = true
			}
		}
	}

	if pkt.IsSeqHDR {
		if pkt.IsVideo {
			stream, err := h264parser.NewCodecDataFromAVCDecoderConfRecord(pkt.Data)
			if err != nil {
				return fmt.Errorf("flv: h264 seqhdr invalid")
			}
			if self.videoidx >= 0 {
				self.streams[self.videoidx] = stream
			} else {
				self.videoidx = len(self.streams)
				self.streams = append(self.streams, stream)
			}
		} else if pkt.IsAudio {
			stream, err := aacparser.NewCodecDataFromMPEG4AudioConfigBytes(pkt.Data)
			if err != nil {
				return fmt.Errorf("flv: aac seqhdr invalid")
			}
			if self.audioidx >= 0 {
				self.streams[self.audioidx] = stream
			} else {
				self.audioidx = len(self.streams)
				self.streams = append(self.streams, stream)
			}
		}
	}

	self.buf.Push(pkt)
	// if pkt.Idx == int8(self.videoidx) && pkt.IsKeyFrame {
	// 	self.curgopcount++
	// }

	// for self.curgopcount >= self.maxgopcount && self.buf.Count > 1 {
	// 	pkt := self.buf.Pop()
	// 	if pkt.Idx == int8(self.videoidx) && pkt.IsKeyFrame {
	// 		self.curgopcount--
	// 	}
	// 	/*
	// 		if pkt.IsSeqHDR {
	// 			self.jumped = true
	// 			log.Println("HDR jumped", self.curgopcount, self.maxgopcount, self.buf.Count)
	// 		}
	// 	*/
	// 	if self.curgopcount < self.maxgopcount {
	// 		break
	// 	}
	// }
	//println("shrink", self.curgopcount, self.maxgopcount, self.buf.Head, self.buf.Tail, "count", self.buf.Count, "size", self.buf.Size)

	self.cond.Broadcast()

	self.lock.Unlock()
	return
}

type QueueCursor struct {
	que    *Queue
	pos    pktque.BufPos
	gotpos bool
	init   func(buf *pktque.Buf, videoidx int) pktque.BufPos
}

func (self *Queue) newCursor() *QueueCursor {
	return &QueueCursor{
		que: self,
	}
}

// Create cursor position at latest packet.
func (self *Queue) Latest() *QueueCursor {
	cursor := self.newCursor()
	cursor.init = func(buf *pktque.Buf, videoidx int) pktque.BufPos {
		return buf.Tail
	}
	return cursor
}

// Create cursor position at oldest buffered packet.
func (self *Queue) Oldest() *QueueCursor {
	cursor := self.newCursor()
	cursor.init = func(buf *pktque.Buf, videoidx int) pktque.BufPos {
		return buf.Head
	}
	return cursor
}

// Create cursor position at specific time in buffered packets.
func (self *Queue) DelayedTime(dur time.Duration) *QueueCursor {
	cursor := self.newCursor()
	cursor.init = func(buf *pktque.Buf, videoidx int) pktque.BufPos {
		i := buf.Tail - 1
		if buf.IsValidPos(i) {
			end := buf.Get(i)
			for buf.IsValidPos(i) {
				if end.Time-buf.Get(i).Time > dur {
					break
				}
				i--
			}
		}
		return i
	}
	return cursor
}

// Create cursor position at specific delayed GOP count in buffered packets.
func (self *Queue) DelayedGopCount(n int) *QueueCursor {
	cursor := self.newCursor()
	cursor.init = func(buf *pktque.Buf, videoidx int) pktque.BufPos {
		i := buf.Tail - 1
		if videoidx != -1 {
			for gop := 0; buf.IsValidPos(i) && gop < n; i-- {
				pkt := buf.Get(i)
				if pkt.Idx == int8(self.videoidx) && pkt.IsKeyFrame {
					gop++
				}
			}
		}
		return i
	}
	return cursor
}

func (self *QueueCursor) Streams() (streams []av.CodecData, err error) {
	self.que.cond.L.Lock()
	for self.que.streams == nil && !self.que.closed {
		self.que.cond.Wait()
	}
	if self.que.streams != nil {
		streams = self.que.streams
	} else {
		err = io.EOF
	}
	self.que.cond.L.Unlock()
	return
}

// ReadPacket will not consume packets in Queue, it's just a cursor.
func (self *QueueCursor) ReadPacket() (pkt av.Packet, err error) {
	self.que.cond.L.Lock()
	buf := self.que.buf
	if !self.gotpos {
		self.pos = self.init(buf, self.que.videoidx)
		self.gotpos = true
	}
	var jumped bool
	for {
		oriPos := self.pos
		if self.pos.LT(buf.Head) {
			self.pos = buf.Head
			jumped = true
			log.Println("less", oriPos, buf.Head)
		} else if self.pos.GT(buf.Tail) {
			self.pos = buf.Tail
			jumped = true
			log.Println("more", oriPos, buf.Tail)
		}
		if buf.IsValidPos(self.pos) {
			pkt = buf.Get(self.pos)
			if jumped && self.que.jumped {
				pkt.Jumped = true
				self.que.jumped = false
			}
			self.pos++
			break
		}
		if self.que.closed {
			err = io.EOF
			break
		}
		self.que.cond.Wait()
	}
	self.que.cond.L.Unlock()
	return
}
