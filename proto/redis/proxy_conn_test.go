package redis

import (
	"overlord/proto"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeBasicOk(t *testing.T) {
	msgs := proto.GetMsgSlice(16)
	data := "*2\r\n$3\r\nGET\r\n$4\r\nbaka\r\n"
	conn := _createConn([]byte(data))
	pc := NewProxyConn(conn)

	nmsgs, err := pc.Decode(msgs)
	assert.NoError(t, err)
	assert.Len(t, nmsgs, 1)
}

func TestDecodeComplexOk(t *testing.T) {
	msgs := proto.GetMsgSlice(16)
	data := "*3\r\n$4\r\nMGET\r\n$4\r\nbaka\r\n$4\r\nkaba\r\n"
	conn := _createConn([]byte(data))
	pc := NewProxyConn(conn)

	nmsgs, err := pc.Decode(msgs)
	assert.NoError(t, err)
	assert.Len(t, nmsgs, 1)
	assert.Len(t, nmsgs[0].Batch(), 2)
}

func TestEncodeCmdOk(t *testing.T) {

	ts := []struct {
		Name   string
		Reps   []*resp
		Obj    *resp
		Expect string
	}{
		{
			Name:   "MergeJoinOk",
			Reps:   []*resp{newRespBulk([]byte("3\r\nabc")), newRespNull(respBulk)},
			Obj:    newRespArray([]*resp{newRespBulk([]byte("4\r\nMGET")), newRespBulk([]byte("3\r\nABC")), newRespBulk([]byte("3\r\nxnc"))}),
			Expect: "*2\r\n$3\r\nabc\r\n$-1\r\n",
		},
		{
			Name: "MergeCountOk",
			Reps: []*resp{newRespInt(1), newRespInt(1), newRespInt(0)},
			Obj: newRespArray(
				[]*resp{
					newRespBulk([]byte("3\r\nDEL")),
					newRespBulk([]byte("1\r\na")),
					newRespBulk([]byte("2\r\nab")),
					newRespBulk([]byte("3\r\nabc")),
				}),
			Expect: ":2\r\n",
		},
		{
			Name: "MergeCountOk",
			Reps: []*resp{newRespString([]byte("OK")), newRespString([]byte("OK"))},
			Obj: newRespArray(
				[]*resp{
					newRespBulk([]byte("4\r\nMSET")),
					newRespBulk([]byte("1\r\na")),
					newRespBulk([]byte("2\r\nab")),
					newRespBulk([]byte("3\r\nabc")),
					newRespBulk([]byte("4\r\nabcd")),
				}),
			Expect: "+OK\r\n",
		},
	}
	for _, tt := range ts {
		t.Run(tt.Name, func(t *testing.T) {
			rs := tt.Reps
			msg := proto.GetMsg()
			co := tt.Obj
			if isComplex(co.nth(0).data) {
				cmds, err := newSubCmd(co)
				if assert.NoError(t, err) {
					for i, cmd := range cmds {
						cmd.reply = rs[i]
						msg.WithRequest(cmd)
					}
					msg.Batch()
				}
			} else {
				cmd := newCommand(co)
				cmd.reply = rs[0]
				msg.WithRequest(cmd)
			}
			data := make([]byte, 2048)
			conn, buf := _createDownStreamConn()
			pc := NewProxyConn(conn)
			err := pc.Encode(msg)
			if assert.NoError(t, err) {
				size, _ := buf.Read(data)
				assert.NoError(t, err)
				assert.Equal(t, tt.Expect, string(data[:size]))
			}
		})
	}

}
