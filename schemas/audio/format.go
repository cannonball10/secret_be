// Package audio holds audio-format constants shared by the tts and stt
// connectors. Values are deliberately strings so they serialize cleanly
// over the wire and map 1:1 onto what most provider APIs expect.
package audio

// AudioFormat names the container + codec of a chunk of audio bytes.
//
// Not every format is supported by every provider; each connector
// documents which subset it can produce or consume.
type AudioFormat string

const (
	// FormatMP3 is MPEG-1 Audio Layer III. Ubiquitous lossy codec,
	// good default for TTS playback in browsers.
	FormatMP3 AudioFormat = "mp3"

	// FormatPCM16 is raw 16-bit little-endian linear PCM. No container.
	// Required by low-latency pipelines (e.g. streaming into WebRTC).
	// Sample rate must be agreed out-of-band.
	FormatPCM16 AudioFormat = "pcm16"

	// FormatOpus is the Opus codec in an Ogg container. Good for
	// real-time audio; supported by ElevenLabs output and browsers.
	FormatOpus AudioFormat = "opus"

	// FormatFLAC is free lossless audio. STT providers often accept it.
	FormatFLAC AudioFormat = "flac"

	// FormatWAV is WAVE container (typically PCM inside). Common STT input.
	FormatWAV AudioFormat = "wav"

	// FormatULaw is G.711 μ-law encoded, 8 kHz. Telephony default.
	FormatULaw AudioFormat = "ulaw"

	// FormatWebM is WebM container, usually with Opus. MediaRecorder
	// default on Chrome — handy for browser-captured STT input.
	FormatWebM AudioFormat = "webm"

	// FormatM4A is MPEG-4 audio (AAC). Common mobile capture output.
	FormatM4A AudioFormat = "m4a"
)

// Common sample rates in Hz. These are informative constants — a
// connector may clamp or reject rates it does not support.
const (
	SampleRate8kHz  = 8000
	SampleRate16kHz = 16000
	SampleRate22kHz = 22050
	SampleRate24kHz = 24000
	SampleRate44kHz = 44100
	SampleRate48kHz = 48000
)

// MIMEType returns the canonical MIME type for the format, or the empty
// string if the format has no single canonical type. Useful when building
// multipart bodies or Content-Type headers.
func (f AudioFormat) MIMEType() string {
	switch f {
	case FormatMP3:
		return "audio/mpeg"
	case FormatPCM16:
		return "audio/pcm"
	case FormatOpus:
		return "audio/ogg"
	case FormatFLAC:
		return "audio/flac"
	case FormatWAV:
		return "audio/wav"
	case FormatULaw:
		return "audio/basic"
	case FormatWebM:
		return "audio/webm"
	case FormatM4A:
		return "audio/mp4"
	default:
		return ""
	}
}
