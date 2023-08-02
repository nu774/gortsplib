package gortsplib

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"

	"github.com/aler9/gortsplib/v2/pkg/media"
	"github.com/aler9/gortsplib/v2/pkg/rtcpsender"
)

type serverStreamMedia struct {
	uuid            uuid.UUID
	media           *media.Media
	formats         map[uint8]*serverStreamFormat
	multicastWriter *serverMulticastWriter
}

func newServerStreamMedia(st *ServerStream, medi *media.Media) *serverStreamMedia {
	sm := &serverStreamMedia{
		uuid:  uuid.New(),
		media: medi,
	}

	sm.formats = make(map[uint8]*serverStreamFormat)
	for _, forma := range medi.Formats {
		tr := &serverStreamFormat{
			format: forma,
		}

		cmedia := medi
		tr.rtcpSender = rtcpsender.New(
			forma.ClockRate(),
			func(pkt rtcp.Packet) {
				st.WritePacketRTCP(cmedia, pkt)
			},
		)

		sm.formats[forma.PayloadType()] = tr
	}

	return sm
}

func (sm *serverStreamMedia) close() {
	for _, tr := range sm.formats {
		if tr.rtcpSender != nil {
			tr.rtcpSender.Close()
		}
	}

	if sm.multicastWriter != nil {
		sm.multicastWriter.close()
	}
}

func (sm *serverStreamMedia) allocateMulticastHandler(s *Server) error {
	if sm.multicastWriter == nil {
		mh, err := newServerMulticastWriter(s)
		if err != nil {
			return err
		}

		sm.multicastWriter = mh
	}
	return nil
}

func (sm *serverStreamMedia) WritePacketRTPWithNTP(ss *ServerStream, pkt *rtp.Packet, ntp time.Time) {
	byts := make([]byte, maxPacketSize)
	n, err := pkt.MarshalTo(byts)
	if err != nil {
		return
	}
	byts = byts[:n]

	forma := sm.formats[pkt.PayloadType]

	forma.rtcpSender.ProcessPacket(pkt, ntp, forma.format.PTSEqualsDTS(pkt))

	// send unicast
	for r := range ss.activeUnicastReaders {
		sm, ok := r.setuppedMedias[sm.media]
		if ok {
			if !sm.writePacketRTP(byts) {
				onWarning(sm.ss, fmt.Errorf("RTP: writer queue is full, pkt dropped: ssrc=%d pt=%d seq=%d ts=%d",
					pkt.SSRC, pkt.PayloadType, pkt.SequenceNumber, pkt.Timestamp))
			}
		}
	}

	// send multicast
	if sm.multicastWriter != nil {
		sm.multicastWriter.writePacketRTP(byts)
	}
}

func (sm *serverStreamMedia) writePacketRTCP(ss *ServerStream, pkt rtcp.Packet) {
	byts, err := pkt.Marshal()
	if err != nil {
		return
	}

	// send unicast
	for r := range ss.activeUnicastReaders {
		sm, ok := r.setuppedMedias[sm.media]
		if ok {
			if !sm.writePacketRTCP(byts) {
				onWarning(sm.ss, fmt.Errorf("RTCP: writer queue is full, pkt dropped"))
			}
		}
	}

	// send multicast
	if sm.multicastWriter != nil {
		sm.multicastWriter.writePacketRTCP(byts)
	}
}
