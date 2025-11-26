package audio

import (
	"encoder/command"
	"encoder/models"
)

// AudioCommand extends the base Command interface with audio-specific operations.
type AudioCommand interface {
	command.Command
	SetCodec(codec string) AudioCommand
	SetBitrate(bitrate string) AudioCommand
	SetSampleRate(rate int) AudioCommand
	SetChannels(channels int) AudioCommand
	SetFilters(filter string) AudioCommand
	SetProgressCallback(callback models.ProgressCallback) AudioCommand
}
