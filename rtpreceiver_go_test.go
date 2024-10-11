// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

//go:build !js
// +build !js

package webrtc

import (
	"context"
	"testing"
	"time"

	"github.com/mincho-artesoft/webrtc/pkg/media"
	"github.com/pion/sdp/v3"
	"github.com/stretchr/testify/assert"
)

func TestSetRTPParameters(t *testing.T) {
	sender, receiver, wan := createVNetPair(t, nil)

	outgoingTrack, err := NewTrackLocalStaticSample(RTPCodecCapability{MimeType: MimeTypeVP8}, "video", "pion")
	assert.NoError(t, err)

	_, err = sender.AddTrack(outgoingTrack)
	assert.NoError(t, err)

	// Those parameters wouldn't make sense in a real application,
	// but for the sake of the test we just need different values.
	p := RTPParameters{
		Codecs: []RTPCodecParameters{
			{
				RTPCodecCapability: RTPCodecCapability{MimeTypeOpus, 48000, 2, "minptime=10;useinbandfec=1", []RTCPFeedback{{"nack", ""}}},
				PayloadType:        111,
			},
		},
		HeaderExtensions: []RTPHeaderExtensionParameter{
			{URI: sdp.SDESMidURI},
			{URI: sdp.SDESRTPStreamIDURI},
			{URI: sdesRepairRTPStreamIDURI},
		},
	}

	seenPacket, seenPacketCancel := context.WithCancel(context.Background())
	receiver.OnTrack(func(_ *TrackRemote, r *RTPReceiver) {
		r.SetRTPParameters(p)

		incomingTrackCodecs := r.Track().Codec()

		assert.EqualValues(t, p.HeaderExtensions, r.Track().params.HeaderExtensions)

		assert.EqualValues(t, p.Codecs[0].MimeType, incomingTrackCodecs.MimeType)
		assert.EqualValues(t, p.Codecs[0].ClockRate, incomingTrackCodecs.ClockRate)
		assert.EqualValues(t, p.Codecs[0].Channels, incomingTrackCodecs.Channels)
		assert.EqualValues(t, p.Codecs[0].SDPFmtpLine, incomingTrackCodecs.SDPFmtpLine)
		assert.EqualValues(t, p.Codecs[0].RTCPFeedback, incomingTrackCodecs.RTCPFeedback)
		assert.EqualValues(t, p.Codecs[0].PayloadType, incomingTrackCodecs.PayloadType)

		seenPacketCancel()
	})

	peerConnectionsConnected := untilConnectionState(PeerConnectionStateConnected, sender, receiver)

	assert.NoError(t, signalPair(sender, receiver))

	peerConnectionsConnected.Wait()
	assert.NoError(t, outgoingTrack.WriteSample(media.Sample{Data: []byte{0xAA}, Duration: time.Second}))

	<-seenPacket.Done()
	assert.NoError(t, wan.Stop())
	closePairNow(t, sender, receiver)
}
