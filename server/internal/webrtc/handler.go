package webrtc

import (
	"bytes"
	"encoding/binary"
	"math"
	"time"

	"github.com/m1k1o/neko/server/internal/webrtc/payload"
	"github.com/m1k1o/neko/server/pkg/types"

	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"
)

func (manager *WebRTCManagerCtx) handle(
	logger zerolog.Logger, data []byte,
	dataChannel *webrtc.DataChannel,
	session types.Session,
) error {
	isHost := session.IsHost()

	//
	// parse header
	//

	buffer := bytes.NewBuffer(data)

	header := &payload.Header{}
	if err := binary.Read(buffer, binary.BigEndian, header); err != nil {
		return err
	}

	//
	// parse body
	//

	// handle cursor move event
	if header.Event == payload.OP_MOVE {
		payload := &payload.Move{}
		if err := binary.Read(buffer, binary.BigEndian, payload); err != nil {
			return err
		}

		x, y := int(payload.X), int(payload.Y)
		if isHost {
			// handle active cursor movement
			manager.desktop.Move(x, y)
			manager.curPosition.Set(x, y)
		} else {
			// handle inactive cursor movement
			session.SetCursor(types.Cursor{
				X: x,
				Y: y,
			})
		}

		return nil
	} else if header.Event == payload.OP_PING {
		ping := &payload.Ping{}
		if err := binary.Read(buffer, binary.BigEndian, ping); err != nil {
			return err
		}

		// create pong header
		header := payload.Header{
			Event:  payload.OP_PONG,
			Length: 19,
		}

		// generate server timestamp
		serverTs := uint64(time.Now().UnixMilli())

		// generate pong payload
		pong := payload.Pong{
			Ping:      *ping,
			ServerTs1: uint32(serverTs / math.MaxUint32),
			ServerTs2: uint32(serverTs % math.MaxUint32),
		}

		buffer := &bytes.Buffer{}

		if err := binary.Write(buffer, binary.BigEndian, header); err != nil {
			return err
		}

		if err := binary.Write(buffer, binary.BigEndian, pong); err != nil {
			return err
		}

		return dataChannel.Send(buffer.Bytes())
	}

	// continue only if session is host
	if !isHost {
		return nil
	}

	switch header.Event {
	case payload.OP_SCROLL:
		// TODO: remove this once the client is fixed
		if header.Length == 4 {
			payload := &payload.Scroll_Old{}
			if err := binary.Read(buffer, binary.BigEndian, payload); err != nil {
				return err
			}

			manager.desktop.Scroll(int(payload.X), int(payload.Y), false)
			logger.Trace().
				Int16("x", payload.X).
				Int16("y", payload.Y).
				Msg("scroll")
		} else {
			payload := &payload.Scroll{}
			if err := binary.Read(buffer, binary.BigEndian, payload); err != nil {
				return err
			}

			manager.desktop.Scroll(int(payload.DeltaX), int(payload.DeltaY), payload.ControlKey)
			logger.Trace().
				Int16("deltaX", payload.DeltaX).
				Int16("deltaY", payload.DeltaY).
				Bool("controlKey", payload.ControlKey).
				Msg("scroll")
		}
	case payload.OP_KEY_DOWN:
		payload := &payload.Key{}
		if err := binary.Read(buffer, binary.BigEndian, payload); err != nil {
			return err
		}

		if err := manager.desktop.KeyDown(payload.Key); err != nil {
			logger.Warn().Err(err).Uint32("key", payload.Key).Msg("key down failed")
		} else {
			logger.Trace().Uint32("key", payload.Key).Msg("key down")
		}
	case payload.OP_KEY_UP:
		payload := &payload.Key{}
		if err := binary.Read(buffer, binary.BigEndian, payload); err != nil {
			return err
		}

		if err := manager.desktop.KeyUp(payload.Key); err != nil {
			logger.Warn().Err(err).Uint32("key", payload.Key).Msg("key up failed")
		} else {
			logger.Trace().Uint32("key", payload.Key).Msg("key up")
		}
	case payload.OP_BTN_DOWN:
		payload := &payload.Key{}
		if err := binary.Read(buffer, binary.BigEndian, payload); err != nil {
			return err
		}

		if err := manager.desktop.ButtonDown(payload.Key); err != nil {
			logger.Warn().Err(err).Uint32("key", payload.Key).Msg("button down failed")
		} else {
			logger.Trace().Uint32("key", payload.Key).Msg("button down")
		}
	case payload.OP_BTN_UP:
		payload := &payload.Key{}
		if err := binary.Read(buffer, binary.BigEndian, payload); err != nil {
			return err
		}

		if err := manager.desktop.ButtonUp(payload.Key); err != nil {
			logger.Warn().Err(err).Uint32("key", payload.Key).Msg("button up failed")
		} else {
			logger.Trace().Uint32("key", payload.Key).Msg("button up")
		}
	case payload.OP_TOUCH_BEGIN:
		payload := &payload.Touch{}
		if err := binary.Read(buffer, binary.BigEndian, payload); err != nil {
			return err
		}

		if err := manager.desktop.TouchBegin(payload.TouchId, int(payload.X), int(payload.Y), payload.Pressure); err != nil {
			logger.Warn().Err(err).Uint32("touchId", payload.TouchId).Msg("touch begin failed")
		} else {
			logger.Trace().Uint32("touchId", payload.TouchId).Msg("touch begin")
		}
	case payload.OP_TOUCH_UPDATE:
		payload := &payload.Touch{}
		if err := binary.Read(buffer, binary.BigEndian, payload); err != nil {
			return err
		}

		if err := manager.desktop.TouchUpdate(payload.TouchId, int(payload.X), int(payload.Y), payload.Pressure); err != nil {
			logger.Warn().Err(err).Uint32("touchId", payload.TouchId).Msg("touch update failed")
		} else {
			logger.Trace().Uint32("touchId", payload.TouchId).Msg("touch update")
		}
	case payload.OP_TOUCH_END:
		payload := &payload.Touch{}
		if err := binary.Read(buffer, binary.BigEndian, payload); err != nil {
			return err
		}

		if err := manager.desktop.TouchEnd(payload.TouchId, int(payload.X), int(payload.Y), payload.Pressure); err != nil {
			logger.Warn().Err(err).Uint32("touchId", payload.TouchId).Msg("touch end failed")
		} else {
			logger.Trace().Uint32("touchId", payload.TouchId).Msg("touch end")
		}
	}

	return nil
}
